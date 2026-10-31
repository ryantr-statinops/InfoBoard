#!/usr/bin/env python3
"""Audit IP-17/IP-08 retention and ownership boundaries on a synthetic database."""

from __future__ import annotations

import argparse
import json
import shutil
import sqlite3
import sys
import tempfile
import time
from pathlib import Path


def read_json(path: Path) -> dict:
    with path.open("r", encoding="utf-8") as handle:
        value = json.load(handle)
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain an object")
    return value


def table_names(conn: sqlite3.Connection) -> set[str]:
    rows = conn.execute("SELECT name FROM sqlite_master WHERE type = 'table'").fetchall()
    return {str(row[0]) for row in rows}


def seed_database(db_path: Path, seed_path: Path, fixture: dict, now: float) -> None:
    if db_path.exists():
        db_path.unlink()
    db_path.parent.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(db_path)
    try:
        conn.executescript(seed_path.read_text(encoding="utf-8"))
        profiles = fixture["profiles"]
        profile_a = str(profiles["profile_a"])
        profile_b = str(profiles["profile_b"])
        now_seconds = int(now)
        now_millis = int(now * 1000)

        conn.execute("INSERT INTO schema_meta(key, value) VALUES ('schema_version', '1')")
        conn.execute(
            "INSERT INTO schema_migrations(version, migration_id, checksum, applied_at) VALUES (1, 'initial_schema_v1', ?, ?)",
            ("0" * 64, now_seconds),
        )
        conn.execute(
            "INSERT INTO configuration(profile_id, persist_activation_recency, updated_at) VALUES (?, 1, ?)",
            (profile_a, now_seconds),
        )
        conn.execute(
            "INSERT INTO configuration(profile_id, persist_activation_recency, updated_at) VALUES (?, 1, ?)",
            (profile_b, now_seconds),
        )
        conn.execute(
            "INSERT INTO installation_state(installation_key, host_version, updated_at) VALUES ('infoboard', '0.1.0', ?)",
            (now_seconds,),
        )
        conn.execute(
            "INSERT INTO browser_state(state_key, state_value) VALUES ('unrelated', 'must-survive')"
        )

        for profile_id, count in ((profile_a, 505), (profile_b, 10)):
            for index in range(count):
                conn.execute(
                    "INSERT INTO activation_metadata(profile_id, context_kind, tab_id, domain, activated_at, source) VALUES (?, 'normal', ?, 'example.test', ?, 'keyboard_enter')",
                    (profile_id, index, now_seconds - index * 60),
                )
        conn.execute(
            "INSERT INTO activation_metadata(profile_id, context_kind, tab_id, domain, activated_at, source) VALUES (?, 'normal', 999, 'example.test', ?, 'keyboard_enter')",
            (profile_a, now_seconds - 31 * 86400),
        )

        payload = "x" * 100_000
        for profile_id in (profile_a, profile_b):
            for index in range(60):
                conn.execute(
                    "INSERT INTO diagnostics(occurred_at, profile_id, event_kind, error_class, retryable, duration_ms, projection_revision, protocol_version, ranking_model_version, schema_version, payload_bytes, redacted_payload) VALUES (?, ?, 'host.health', 'none', 0, 0, 0, 1, 'rank-1', 1, ?, ?)",
                    (now_millis - index * 60_000, profile_id, len(payload), payload),
                )
        conn.execute(
            "INSERT INTO diagnostics(occurred_at, profile_id, event_kind, error_class, retryable, duration_ms, projection_revision, protocol_version, ranking_model_version, schema_version, payload_bytes, redacted_payload) VALUES (?, ?, 'host.health', 'none', 0, 0, 0, 1, 'rank-1', 1, 1, 'x')",
            (now_millis - 8 * 86400 * 1000, profile_a),
        )
        conn.commit()
    finally:
        conn.close()


def apply_retention(conn: sqlite3.Connection, fixture: dict, now: float) -> None:
    limits = fixture["limits"]
    activation_cutoff = int(now) - int(limits["activation_age_days"]) * 86400
    diagnostics_cutoff = int(now * 1000) - int(limits["diagnostics_age_days"]) * 86400 * 1000
    conn.execute("DELETE FROM activation_metadata WHERE activated_at < ?", (activation_cutoff,))
    conn.execute(
        """DELETE FROM activation_metadata WHERE activation_id IN (
            SELECT activation_id FROM (
                SELECT activation_id, ROW_NUMBER() OVER (
                    PARTITION BY profile_id ORDER BY activated_at DESC, activation_id DESC
                ) AS retention_rank FROM activation_metadata
            ) WHERE retention_rank > ?
        )""",
        (int(limits["activation_rows_per_profile"]),),
    )
    conn.execute("DELETE FROM diagnostics WHERE occurred_at < ?", (diagnostics_cutoff,))
    total = int(conn.execute("SELECT coalesce(sum(payload_bytes), 0) FROM diagnostics").fetchone()[0])
    max_bytes = int(limits["diagnostics_bytes"])
    if total > max_bytes:
        conn.execute(
            """DELETE FROM diagnostics WHERE diagnostic_id IN (
                SELECT diagnostic_id FROM (
                    SELECT diagnostic_id,
                        SUM(payload_bytes) OVER (
                            ORDER BY occurred_at DESC, diagnostic_id DESC
                            ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
                        ) AS retained_bytes
                    FROM diagnostics
                ) WHERE retained_bytes > ?
            )""",
            (max_bytes,),
        )
    conn.commit()


def probe_private_insert(conn: sqlite3.Connection, fixture: dict) -> list[str]:
    findings: list[str] = []
    profile = str(fixture["profiles"]["profile_a"])
    try:
        conn.execute(
            "INSERT INTO activation_metadata(profile_id, context_kind, tab_id, domain, activated_at, source) VALUES (?, 'private', 1, 'example.test', 1, 'keyboard_enter')",
            (profile,),
        )
        conn.rollback()
        findings.append("private_activation_accepted")
    except sqlite3.IntegrityError:
        conn.rollback()
    return findings


def probe_reset(db_path: Path, fixture: dict) -> list[str]:
    findings: list[str] = []
    profile_a = str(fixture["profiles"]["profile_a"])
    profile_b = str(fixture["profiles"]["profile_b"])
    with tempfile.TemporaryDirectory() as temp_dir:
        copy_path = Path(temp_dir) / "reset.sqlite"
        shutil.copyfile(db_path, copy_path)
        conn = sqlite3.connect(copy_path)
        try:
            for table in ("configuration", "activation_metadata", "diagnostics"):
                conn.execute(f"DELETE FROM {table} WHERE profile_id = ?", (profile_a,))
            conn.commit()
            for table in ("configuration", "activation_metadata", "diagnostics"):
                remaining_a = conn.execute(f"SELECT count(*) FROM {table} WHERE profile_id = ?", (profile_a,)).fetchone()[0]
                remaining_b = conn.execute(f"SELECT count(*) FROM {table} WHERE profile_id = ?", (profile_b,)).fetchone()[0]
                if remaining_a != 0:
                    findings.append(f"reset_left_profile_a_rows:{table}")
                if remaining_b == 0:
                    findings.append(f"reset_removed_profile_b_rows:{table}")
            installation_rows = conn.execute("SELECT count(*) FROM installation_state").fetchone()[0]
            if installation_rows != 1:
                findings.append("reset_removed_installation_state")
        finally:
            conn.close()
    return findings


def probe_uninstall(db_path: Path, fixture: dict) -> list[str]:
    findings: list[str] = []
    owned = set(fixture["expected"]["owned_tables"])
    unrelated = set(fixture["expected"]["unrelated_tables"])
    with tempfile.TemporaryDirectory() as temp_dir:
        copy_path = Path(temp_dir) / "uninstall.sqlite"
        shutil.copyfile(db_path, copy_path)
        conn = sqlite3.connect(copy_path)
        try:
            for table in sorted(owned):
                conn.execute(f"DROP TABLE IF EXISTS {table}")
            conn.commit()
            remaining = table_names(conn)
            for table in owned:
                if table in remaining:
                    findings.append(f"uninstall_left_owned_table:{table}")
            for table in unrelated:
                if table not in remaining:
                    findings.append(f"uninstall_removed_unrelated_table:{table}")
        finally:
            conn.close()
    return findings


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--db", type=Path, required=True)
    parser.add_argument("--fixtures", type=Path, default=Path("fixtures/privacy/phase-17"))
    parser.add_argument("--max-age-days", type=int, default=30)
    parser.add_argument("--max-diagnostics-bytes", type=int, default=10_000_000)
    parser.add_argument("--json", action="store_true")
    parser.add_argument("--keep-db", action="store_true")
    args = parser.parse_args()

    fixture = read_json(args.fixtures / "retention-audit.json")
    if args.max_age_days != int(fixture["limits"]["activation_age_days"]):
        print("FAIL: --max-age-days does not match the fixture contract", file=sys.stderr)
        return 1
    if args.max_diagnostics_bytes != int(fixture["limits"]["diagnostics_bytes"]):
        print("FAIL: --max-diagnostics-bytes does not match the fixture contract", file=sys.stderr)
        return 1

    now = time.time()
    created = not args.db.exists()
    if created:
        seed_database(args.db, args.fixtures / str(fixture["seed"]), fixture, now)

    conn = sqlite3.connect(args.db)
    findings: list[str] = []
    counts = {"profile_a": 0, "profile_b": 0}
    diagnostics_rows = 0
    diagnostics_bytes = 0
    owned = set(fixture["expected"]["owned_tables"])
    try:
        apply_retention(conn, fixture, now)
        counts = {
            "profile_a": int(conn.execute("SELECT count(*) FROM activation_metadata WHERE profile_id = ?", (fixture["profiles"]["profile_a"],)).fetchone()[0]),
            "profile_b": int(conn.execute("SELECT count(*) FROM activation_metadata WHERE profile_id = ?", (fixture["profiles"]["profile_b"],)).fetchone()[0]),
        }
        expected_counts = fixture["expected"]["activation_counts_after"]
        if counts["profile_a"] != int(expected_counts["profile_a"]):
            findings.append(f"activation_count_profile_a:{counts['profile_a']}")
        if counts["profile_b"] != int(expected_counts["profile_b"]):
            findings.append(f"activation_count_profile_b:{counts['profile_b']}")
        if int(conn.execute("SELECT count(*) FROM activation_metadata WHERE context_kind <> 'normal'").fetchone()[0]) != 0:
            findings.append("private_activation_rows_present")

        activation_cutoff = int(now) - int(fixture["limits"]["activation_age_days"]) * 86400
        diagnostics_cutoff = int(now * 1000) - int(fixture["limits"]["diagnostics_age_days"]) * 86400 * 1000
        if int(conn.execute("SELECT count(*) FROM activation_metadata WHERE activated_at < ?", (activation_cutoff,)).fetchone()[0]) != 0:
            findings.append("old_activation_rows_present")
        if int(conn.execute("SELECT count(*) FROM diagnostics WHERE occurred_at < ?", (diagnostics_cutoff,)).fetchone()[0]) != 0:
            findings.append("old_diagnostics_rows_present")

        diagnostics_bytes = int(conn.execute("SELECT coalesce(sum(payload_bytes), 0) FROM diagnostics").fetchone()[0])
        diagnostics_rows = int(conn.execute("SELECT count(*) FROM diagnostics").fetchone()[0])
        if diagnostics_bytes > int(fixture["limits"]["diagnostics_bytes"]):
            findings.append(f"diagnostics_bytes:{diagnostics_bytes}")

        owned = set(fixture["expected"]["owned_tables"])
        present = table_names(conn)
        for table in owned:
            if table not in present:
                findings.append(f"missing_owned_table:{table}")

        findings.extend(probe_private_insert(conn, fixture))
    finally:
        conn.close()

    findings.extend(probe_reset(args.db, fixture))
    findings.extend(probe_uninstall(args.db, fixture))

    report = {
        "ok": not findings,
        "findings": findings,
        "activation_counts": counts,
        "diagnostics_rows": diagnostics_rows,
        "diagnostics_bytes": diagnostics_bytes,
        "activation_age_days": int(fixture["limits"]["activation_age_days"]),
        "diagnostics_age_days": int(fixture["limits"]["diagnostics_age_days"]),
        "diagnostics_scope": "global",
        "owned_tables": sorted(owned),
        "unrelated_tables": sorted(fixture["expected"]["unrelated_tables"]),
    }
    if created and not args.keep_db:
        args.db.unlink(missing_ok=True)
    if args.json:
        print(json.dumps(report, indent=2, sort_keys=True))
    elif findings:
        print("FAIL: " + ", ".join(findings), file=sys.stderr)
    else:
        print(
            f"PASS: activation {counts['profile_a']}/{counts['profile_b']}, "
            f"diagnostics {diagnostics_rows} rows / {diagnostics_bytes} bytes, "
            "private rows absent, reset profile-scoped, uninstall owned tables only"
        )
    return 0 if not findings else 1


if __name__ == "__main__":
    raise SystemExit(main())
