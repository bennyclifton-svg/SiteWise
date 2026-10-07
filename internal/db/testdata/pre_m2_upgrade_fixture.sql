-- Synthetic data only. Run exclusively in the newly created rehearsal database.
DO $$
DECLARE n integer; tenant integer; o uuid; p uuid; s uuid; part uuid; w uuid; proposal text;
BEGIN
 FOR n IN 1..2 LOOP
  o:=md5('upgrade-org'||n)::uuid; p:=md5('upgrade-project'||n)::uuid; s:=md5('upgrade-site'||n)::uuid;
  INSERT INTO orgs(id,name) VALUES(o,'Synthetic upgrade tenant '||n);
  INSERT INTO users(org_id,id,email) VALUES(o,md5('upgrade-user'||n)::uuid,'synthetic'||n||'@example.invalid');
  INSERT INTO sites(org_id,id,label) VALUES(o,s,'Synthetic site');
  INSERT INTO projects(org_id,id,name,site_id) VALUES(o,p,'Synthetic upgrade project',s);
  INSERT INTO project_parts(org_id,id,created_by_project_id,site_id,label,kind)
   VALUES(o,md5('upgrade-part'||n)::uuid,p,s,'Whole','whole');
  INSERT INTO packages(org_id,id,project_id,kind,title,lifecycle_status,origin,review_status)
   VALUES(o,md5('upgrade-package'||n)::uuid,p,'services','Synthetic appointment','planned','user','accepted_for_planning');
  INSERT INTO package_stages(org_id,id,project_id,package_id,stage_id,label,ordinal,origin)
   VALUES(o,md5('upgrade-stage'||n)::uuid,p,md5('upgrade-package'||n)::uuid,'design','Design',1,'user');
  INSERT INTO reports(org_id,id,project_id,kind,title) VALUES(o,md5('upgrade-report'||n)::uuid,p,'pmp','Synthetic draft');
  INSERT INTO report_versions(org_id,id,report_id,project_id,number,status,reporting_date,source_revisions,template_id,template_version,sections)
   VALUES(o,md5('upgrade-version'||n)::uuid,md5('upgrade-report'||n)::uuid,p,1,'draft','2026-10-07','{}','tpl.pmp',1,'[]');
  UPDATE reports SET current_draft_version_id=md5('upgrade-version'||n)::uuid WHERE org_id=o;
 END LOOP;
 FOR n IN 1..3 LOOP
  tenant:=CASE WHEN n=3 THEN 2 ELSE 1 END;
  o:=md5('upgrade-org'||tenant)::uuid;p:=md5('upgrade-project'||tenant)::uuid;s:=md5('upgrade-site'||tenant)::uuid;
  part:=md5('upgrade-part'||tenant)::uuid;w:=md5('upgrade-work'||n)::uuid;proposal:='ic.upgrade'||n||'||||0';
  INSERT INTO files(org_id,id,project_id,sha256,byte_size,media_type)
   VALUES(o,md5('upgrade-file'||n)::uuid,p,decode(repeat(lpad(to_hex(n),2,'0'),32),'hex'),30,'text/plain');
  INSERT INTO documents(org_id,id,project_id,file_id,filename,status,document_number,revision)
   VALUES(o,md5('upgrade-doc'||n)::uuid,p,md5('upgrade-file'||n)::uuid,'Synthetic source '||n||'.txt','filed','SYN-'||n,'A');
  INSERT INTO passages(org_id,id,document_id,ordinal,body)
   VALUES(o,md5('upgrade-passage'||n)::uuid,md5('upgrade-doc'||n)::uuid,1,'Synthetic source excerpt');
  INSERT INTO passage_sources(org_id,passage_id,page,location,outcome,confidence,mapped_keys)
   VALUES(o,md5('upgrade-passage'||n)::uuid,1,'Synthetic paragraph',CASE n WHEN 1 THEN 'mapped' WHEN 2 THEN 'needs_mapping' ELSE 'pending' END,0.91,ARRAY['det.synthetic']);
  INSERT INTO passage_calls(org_id,passage_id,stage,fingerprint,result)
   VALUES(o,md5('upgrade-passage'||n)::uuid,'evidence','synthetic-recorded','{"synthetic":true}');
  INSERT INTO profile_rows(org_id,project_id,site_id,part_id,key,value,band,origin,review_status,meaning,value_state,sources,alternatives)
   VALUES(o,p,s,part,'det.synthetic'||n,'unknown','user','assumption','accepted_for_planning','stated','unknown',jsonb_build_array(jsonb_build_object('document_id',md5('upgrade-doc'||n)::uuid,'revision','A')),'[]');
  INSERT INTO work_items(org_id,id,project_id,site_id,part_id,system_id,action,inclusion,title,origin,review_status,user_touched,provenance)
   VALUES(o,w,p,s,part,'hydraulic.gas','repair','included','Synthetic work '||n,'user','accepted_for_planning',true,jsonb_build_object('actor',md5('upgrade-user'||tenant)::uuid));
  INSERT INTO package_scope_items(org_id,id,project_id,package_id,item_kind,work_item_id,role,user_text,inclusion,origin,review_status)
   VALUES(o,md5('upgrade-scope'||n)::uuid,p,md5('upgrade-package'||tenant)::uuid,'responsibility',w,'design','Synthetic design scope','included','user','accepted_for_planning');
  INSERT INTO project_delivery_items(org_id,id,project_id,kind,title,status,package_id,work_item_id,target_date,origin,review_status)
   VALUES(o,md5('upgrade-delivery'||n)::uuid,p,'activity','Synthetic delivery '||n,'not_started',md5('upgrade-package'||tenant)::uuid,w,'2026-11-01','user','accepted_for_planning');
  INSERT INTO proposals(org_id,project_id,site_id,key,record_kind,record_id,proposal_index,kind,label,reason,severity,specificity,rank,draft,unaccepted_triggers,inputs_fingerprint,knowledge_version,state)
   VALUES(o,p,s,proposal,'ic','ic.upgrade'||n,0,'investigation','Synthetic review '||n,'{}',CASE WHEN n=1 THEN 'life-safety' ELSE 'compliance' END,1,CASE WHEN n=2 THEN 2 ELSE 1 END,true,false,repeat('a',64),'synthetic-old','dismissed');
  INSERT INTO proposal_triggers(org_id,project_id,proposal_key,work_item_id) VALUES(o,p,proposal,w);
  INSERT INTO proposal_decisions(org_id,id,project_id,proposal_key,record_id,trigger_work_item_id,decision,inputs_fingerprint,actor,rationale)
   VALUES(o,md5('upgrade-decision'||n)::uuid,p,proposal,'ic.upgrade'||n,w,'dismissed',repeat('a',64),md5('upgrade-user'||tenant)::uuid,'Synthetic retained decision');
 END LOOP;
 INSERT INTO delivery_dependencies(org_id,project_id,predecessor_id,successor_id,lag_days)
  VALUES(md5('upgrade-org1')::uuid,md5('upgrade-project1')::uuid,md5('upgrade-delivery1')::uuid,md5('upgrade-delivery2')::uuid,2);
END;
$$;
