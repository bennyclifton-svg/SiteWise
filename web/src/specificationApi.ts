import { request } from "./api";

export interface SpecificationType {
	 authority?: string; source_document_id?: string; starter_key?: string; suggested_type_id?: string;
	 origin?: string; review_status?: string;
  id: string; project_id: string; code: string; title: string; kind: string; system_id: string;
  description: string; scheme: string; option_group: string; status: string; version: number;
}
export interface SpecificationApplication {
  id: string; project_id: string; type_id: string; part_id?: string; work_item_id?: string;
  location: string; surface: string; action: string; inclusion: string; selection: string;
  quantity?: number; unit?: string; version: number;
}
export interface SpecificationValue { state: string; text?: string; number?: number; upper?: number; boolean?: boolean; unit?: string; qualifier?: string; basis?: string; method?: string; datum?: string }
export interface SpecificationSource { document_id?: string; hash?: string; page?: number; location?: string; excerpt?: string; issue?: string; marker?: string; unavailable?: boolean; stale?: boolean }
export interface SpecificationAssertion {
  id: string; type_id: string; application_id?: string; key: string; value: SpecificationValue; meaning: string;
  source: SpecificationSource; origin: string; status: string; actor: string; reason: string; version: number; created_at: string;
}
export interface SpecificationRelationship { id: string; from_id: string; to_id: string; kind: string; sequence?: number; proportion?: number; basis?: string; note?: string; version: number }
export interface SpecificationReference { document_id: string; filename: string; number: string; revision: string; hash: string; location: string; unavailable?: boolean; stale?: boolean }
export interface SpecificationCoverage { id: string; system_id: string; type_id?: string; mode: string; extent: string; exclusions: string; maintainer: string; reviewer: string; references: SpecificationReference[]; reason: string; version: number }
export interface SpecificationScope { work_item_id: string; system_id: string; part_id: string; location: string; title: string; action: string; inclusion: string; provider: string; retired: boolean; version: number }
export interface SpecificationStarter { key: string; code: string; title: string; system_id: string; kind: string }
export interface SpecificationSection { id: string; label: string; systems: {id:string;label:string}[]; starters: SpecificationStarter[] }
export interface SpecificationSnapshot { project_id: string; project_name: string; revision: number; types: SpecificationType[]; applications: SpecificationApplication[]; assertions: SpecificationAssertion[]; relationships: SpecificationRelationship[]; coverage: SpecificationCoverage[]; scope: SpecificationScope[] }
export interface SpecificationAttribute { key: string; label: string; group: string; type: string; units?: string[]; version: number }
export interface SpecificationRow { authority: string; type_id: string; application_id?: string; code: string; title: string; system_id: string; location: string; surface: string; action: string; selection: string; inclusion: string; scheme: string; option_group: string; description: string; pending: number; relationships: string[]; fields: { key: string; label: string; meaning: string; value: string; source: SpecificationSource }[] }
export interface SpecificationView { snapshot: SpecificationSnapshot; sections: SpecificationSection[]; attributes: SpecificationAttribute[]; systems: { id: string; label: string }[]; rows: SpecificationRow[]; imports: { document_id: string; status: string; count: number; error: string }[] }
export interface SpecificationIssue { id: string; number: number; title: string; purpose: string; created_at: string; snapshot: SpecificationSnapshot }
const base = (p: string) => `/projects/${p}/specifications`;
export const specificationApi = {
  update: (p:string) => request<{queued:number}>("POST",`${base(p)}/update`,{}),
  coverage: (p:string,value:SpecificationCoverage) => request<SpecificationCoverage>("POST",`${base(p)}/coverage`,value),
  adopt: (p:string,t:SpecificationType,reason:string) => request<SpecificationType>("POST",`${base(p)}/types/${t.id}/adopt`,{version:t.version,reason}),
  link: (p:string,t:SpecificationType,target_type_id:string) => request("POST",`${base(p)}/types/${t.id}/link`,{version:t.version,target_type_id}),
  cell: (p:string,type_id:string,key:string,value:SpecificationValue,meaning:string,expected_accepted_id:string) => request<SpecificationAssertion>("POST",`${base(p)}/cells`,{type_id,key,value,meaning,expected_accepted_id}),
	 import: (p: string, document_id: string, target_type_id: string) => request<{status:string}>("POST",`${base(p)}/imports`,{document_id,target_type_id}),
  get: (p: string) => request<SpecificationView>("GET", base(p)),
  saveType: (p: string, value: SpecificationType) => request<SpecificationType>(value.id ? "PUT" : "POST", `${base(p)}/types${value.id ? `/${value.id}` : ""}`, value),
  saveApplication: (p: string, value: SpecificationApplication) => request<SpecificationApplication>(value.id ? "PUT" : "POST", `${base(p)}/applications${value.id ? `/${value.id}` : ""}`, value),
  addAssertion: (p: string, value: Partial<SpecificationAssertion>) => request<SpecificationAssertion>("POST", `${base(p)}/assertions`, value),
  review: (p: string, a: SpecificationAssertion, status: string, prior: string, reason: string, sourceReviewed: boolean) => request<SpecificationAssertion>("POST", `${base(p)}/assertions/${a.id}/review`, { version: a.version, status, expected_accepted_id: prior, reason, source_reviewed: sourceReviewed }),
  relationship: (p: string, value: Partial<SpecificationRelationship>) => request<SpecificationRelationship>("POST", `${base(p)}/relationships`, value),
  issues: (p: string) => request<SpecificationIssue[]>("GET", `${base(p)}/issues`),
  issue: (p: string, title: string, purpose: string, revision: number) => request<SpecificationIssue>("POST", `${base(p)}/issues`, {title,purpose,revision}),
  exportURL: (p: string, issue?: string) => `/api${base(p)}${issue ? `/issues/${issue}` : ""}/export`,
};
