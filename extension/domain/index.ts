export type ProfileID = string & { readonly __profileID: unique symbol };
export type ContextKind = 'normal' | 'private';
export type ProjectionEpoch = string & { readonly __projectionEpoch: unique symbol };
export type ProjectionRevision = number & { readonly __projectionRevision: unique symbol };

const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
export function parseProfileID(value: unknown): ProfileID {
  if (typeof value !== 'string' || value.length !== 36 || !UUID_V4.test(value)) throw new TypeError('INVALID_PROFILE_ID');
  return value as ProfileID;
}
export function parseProjectionEpoch(value: unknown): ProjectionEpoch {
  if (typeof value !== 'string' || value.length !== 36 || !UUID_V4.test(value)) throw new TypeError('INVALID_PROJECTION_EPOCH');
  return value as ProjectionEpoch;
}
export function parseRevision(value: unknown): ProjectionRevision {
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < 0 || value > Number.MAX_SAFE_INTEGER) throw new TypeError('INVALID_PROJECTION_REVISION');
  return value as ProjectionRevision;
}
export function createProfileID(): ProfileID { return parseProfileID(crypto.randomUUID()); }
export function createProjectionEpoch(): ProjectionEpoch { return parseProjectionEpoch(crypto.randomUUID()); }
export interface TabIdentity { profile_id: ProfileID; context_kind: ContextKind; tab_id: number }
export function tabIdentityKey(id: TabIdentity): string { return JSON.stringify([id.profile_id, id.context_kind, id.tab_id]); }
export interface EligibleTabRecord {
 profile_id: ProfileID; context_kind: ContextKind; tab_identity: TabIdentity; window_id: number; group_id: number | null;
 title_display: string; title_search: string; url_search: string; url_display: string; domain_display: string; domain_search: string;
 window_label_display?: string; window_label_search?: string; group_label_display?: string; group_label_search?: string;
 pinned: boolean; active: boolean; eligible: boolean; observed_at: number; projection_epoch: ProjectionEpoch; projection_revision: ProjectionRevision;
}
export interface ProjectionState { profile_id: ProfileID; context_kind: ContextKind; epoch: ProjectionEpoch | null; revision: ProjectionRevision; records: Map<string, EligibleTabRecord>; sequences: Map<number, string> }
export type ProjectionOutcome = 'APPLIED' | 'DUPLICATE' | 'SNAPSHOT_REQUIRED' | 'REVISION_MISMATCH';
export function emptyProjection(profile_id: ProfileID, context_kind: ContextKind): ProjectionState { return {profile_id,context_kind,epoch:null,revision:parseRevision(0),records:new Map(),sequences:new Map()}; }
const boundedString=(v:unknown,max:number,scalars=false): v is string => typeof v === 'string' && (scalars ? [...v].length : new TextEncoder().encode(v).length) <= max;
export function validateTab(record: EligibleTabRecord, profile: ProfileID, context: ContextKind, epoch: ProjectionEpoch): boolean {
 const i=record.tab_identity; return record.profile_id===profile && record.context_kind===context && i.profile_id===profile && i.context_kind===context && Number.isSafeInteger(i.tab_id) && i.tab_id>=0 && Number.isSafeInteger(record.window_id) && record.window_id>=0 && (record.group_id===null || (Number.isSafeInteger(record.group_id)&&record.group_id>=0)) && boundedString(record.title_display,512,true) && boundedString(record.title_search,512,true) && boundedString(record.url_search,2048) && boundedString(record.url_display,2048) && boundedString(record.domain_display,255) && boundedString(record.domain_search,255) && typeof record.pinned==='boolean' && typeof record.active==='boolean' && record.eligible===true && record.projection_epoch===epoch && Number.isSafeInteger(record.observed_at);
}
export function acceptSnapshot(state: ProjectionState, epoch: ProjectionEpoch, rows: EligibleTabRecord[]): ProjectionOutcome {
 if (rows.length>10000) return 'SNAPSHOT_REQUIRED'; const next=new Map<string,EligibleTabRecord>();
 for (const row of rows) { if(!validateTab(row,state.profile_id,state.context_kind,epoch)) return 'SNAPSHOT_REQUIRED'; const key=tabIdentityKey(row.tab_identity); if(next.has(key)) return 'SNAPSHOT_REQUIRED'; next.set(key,{...row}); }
 state.epoch=epoch; state.revision=parseRevision(1); state.records=next; state.sequences.clear(); return 'APPLIED';
}
export interface ProjectionEvent { profile_id: ProfileID; context_kind: ContextKind; epoch: ProjectionEpoch; sequence: number; previous_revision: number; operation:'upsert'|'remove'; tab_identity:TabIdentity; record?:EligibleTabRecord }
export function applyEvent(state:ProjectionState,event:ProjectionEvent):ProjectionOutcome {
 if(event.profile_id!==state.profile_id||event.context_kind!==state.context_kind||!state.epoch||event.epoch!==state.epoch)return 'SNAPSHOT_REQUIRED';
 const key=tabIdentityKey(event.tab_identity), old=state.sequences.get(event.sequence), signature=JSON.stringify(event); if(old!==undefined)return old===signature?'DUPLICATE':'REVISION_MISMATCH';
 if(event.sequence!==Number(state.revision)+1||event.previous_revision!==Number(state.revision))return 'SNAPSHOT_REQUIRED';
 if(event.operation==='upsert'&&(!event.record||!validateTab(event.record,state.profile_id,state.context_kind,state.epoch)||tabIdentityKey(event.record.tab_identity)!==key))return 'SNAPSHOT_REQUIRED';
 if(event.operation==='remove'&&!state.records.has(key))return 'SNAPSHOT_REQUIRED';
 const next=new Map(state.records); if(event.operation==='remove')next.delete(key);else next.set(key,{...event.record!}); state.records=next;state.revision=parseRevision(Number(state.revision)+1);state.sequences.set(event.sequence,signature);return 'APPLIED';
}
export interface ProfileIDStore { read():Promise<unknown|null>; write(id:ProfileID):Promise<void>; removeAfterDisconnect():Promise<void> }
export async function loadProfileID(store:ProfileIDStore):Promise<ProfileID> { const raw=await store.read(); if(raw===null){const id=createProfileID();await store.write(id);return id;} return parseProfileID(raw); }
export interface ActivationReference {profile_id:ProfileID;context_kind:ContextKind;tab_id:number;projection_epoch:ProjectionEpoch;projection_revision:ProjectionRevision;result_id:string}
export interface ActivationMetadata {profile_id:ProfileID;context_kind:'normal';tab_identity:TabIdentity;domain:string;activated_at:number;source:'keyboard_enter'|'surface_select';storage_sequence:number}
export function retainActivations(rows:ActivationMetadata[],now:number):ActivationMetadata[] {return rows.filter(x=>x.context_kind==='normal'&&x.activated_at>=now-30*86400000&&x.activated_at<=now).sort((a,b)=>a.activated_at-b.activated_at||a.storage_sequence-b.storage_sequence).slice(-500);}
