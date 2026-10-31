"""Release compatibility tuple checks shared by packaging gates."""

from __future__ import annotations

from typing import Any


def release_tuple_errors(release: dict[str, Any]) -> list[str]:
    errors: list[str] = []
    compatibility = release.get("compatibility")
    if not isinstance(compatibility, dict):
        return ["compatibility tuple is missing"]
    protocol = release.get("protocol", {}).get("major")
    if protocol not in compatibility.get("protocol_major", []):
        errors.append("release protocol major is not supported")
    if release.get("ranking_model_version") not in compatibility.get("ranking_model_versions", []):
        errors.append("ranking model version is not supported")
    if release.get("storage_schema_version") not in compatibility.get("storage_schema_versions", []):
        errors.append("storage schema version is not supported")
    versions = release.get("versions", {})
    if versions.get("product") not in compatibility.get("product_versions", []):
        errors.append("product version is not supported")
    if versions.get("host") not in compatibility.get("host_versions", []):
        errors.append("host version is not supported")
    if versions.get("chrome") != versions.get("product"):
        errors.append("chrome version differs from the product version")
    if versions.get("edge") != versions.get("product"):
        errors.append("edge version differs from the product version")
    return errors
