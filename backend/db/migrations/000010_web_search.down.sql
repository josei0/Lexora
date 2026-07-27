drop table web_searches;
alter table plans drop column web_search_enabled;
alter table plans drop column daily_web_searches;
alter table plans drop column daily_messages;
alter table citations drop column source_url;
drop index idx_documents_source_url;
alter table documents drop column source_url;
