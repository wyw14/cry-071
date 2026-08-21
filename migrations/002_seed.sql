BEGIN;
INSERT INTO public_areas(id,name,district,active) VALUES
('area-central-park','中心公园','东城区',true),
('area-riverside','滨河步道','西城区',true)
ON CONFLICT(id) DO NOTHING;

INSERT INTO facility_categories(id,name,description,active) VALUES
('lighting','照明设施','路灯与景观照明',true),
('sanitation','环卫设施','垃圾桶与清洁设施',true),
('accessibility','无障碍设施','坡道、扶手与盲道',true)
ON CONFLICT(id) DO NOTHING;

INSERT INTO feedback_subjects(code,name,default_priority,default_assignee_id) VALUES
('damaged','设施损坏','high','agent-maintenance'),
('cleanliness','环境卫生','normal','agent-sanitation'),
('safety','安全隐患','critical','agent-safety')
ON CONFLICT(code) DO NOTHING;
COMMIT;
