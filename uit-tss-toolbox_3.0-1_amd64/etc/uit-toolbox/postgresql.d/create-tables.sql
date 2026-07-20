CREATE TABLE IF NOT EXISTS ids (
	uuid UUID PRIMARY KEY DEFAULT uuidv7(),
	tagnumber INTEGER NOT NULL UNIQUE,
	system_serial VARCHAR(128) NOT NULL UNIQUE,
	time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,

	CONSTRAINT ids_valid_tag
		CHECK (tagnumber > 100000 AND tagnumber < 999999),
	CONSTRAINT ids_system_serial_format
		CHECK (system_serial ~ '^[a-zA-Z0-9_-]+$'),
	CONSTRAINT ids_system_serial_length
		CHECK (CHAR_LENGTH(system_serial) >= 1 AND CHAR_LENGTH(system_serial) <= 128)
);

CREATE TABLE IF NOT EXISTS serverstats (
	date DATE UNIQUE NOT NULL,
	client_count SMALLINT DEFAULT NULL,
	total_os_installed SMALLINT DEFAULT NULL,
	battery_health DECIMAL(5,2) DEFAULT NULL,
	disk_health DECIMAL(5,2) DEFAULT NULL,
	total_job_count SMALLINT DEFAULT NULL,
	clone_job_count SMALLINT DEFAULT NULL,
	erase_job_count SMALLINT DEFAULT NULL,
	avg_clone_time SMALLINT DEFAULT NULL,
	avg_erase_time SMALLINT DEFAULT NULL,
	last_image_update DATE DEFAULT NULL
);


CREATE TABLE IF NOT EXISTS jobstats (
	uuid UUID PRIMARY KEY,
	client_uuid UUID NOT NULL,
	tagnumber INTEGER DEFAULT NULL, -- unused, moved to ids table
	time TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	system_serial VARCHAR(128) DEFAULT NULL,
	disk_name VARCHAR(32) DEFAULT NULL,
	avg_cpu_temp SMALLINT DEFAULT NULL,
	avg_cpu_usage DECIMAL(6,2) DEFAULT NULL,
	avg_network_usage DECIMAL(5,2) DEFAULT NULL,
	erase_completed BOOLEAN DEFAULT FALSE,
	erase_mode VARCHAR(24) DEFAULT NULL,
	erase_diskpercent SMALLINT DEFAULT NULL,
	erase_time SMALLINT DEFAULT NULL,
	clone_completed BOOLEAN DEFAULT FALSE,
	clone_image VARCHAR(36) DEFAULT NULL,
	clone_master BOOLEAN DEFAULT FALSE,
	clone_time SMALLINT DEFAULT NULL,
	job_cancelled BOOLEAN DEFAULT FALSE,

	CONSTRAINT jobstats_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_jobstats_time ON jobstats (time DESC NULLS LAST);

CREATE TABLE IF NOT EXISTS historical_hardware_data (
	transaction_uuid UUID PRIMARY KEY,
	time TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	client_uuid UUID NOT NULL,
	ethernet_mac VARCHAR(17) DEFAULT NULL, -- has to be migrated to hardware_data
	wifi_mac VARCHAR(17) DEFAULT NULL, -- has to be migrated to hardware_data
	memory_serial TEXT[] DEFAULT NULL,
	memory_capacity_kb BIGINT DEFAULT NULL,
	memory_speed_mhz SMALLINT DEFAULT NULL,
	updated_from_windows BOOLEAN DEFAULT FALSE NOT NULL,

	CONSTRAINT historical_hardware_data_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_historical_hardware_data_time ON historical_hardware_data (time DESC NULLS LAST);

CREATE TABLE IF NOT EXISTS locations (
	client_uuid UUID PRIMARY KEY,
	time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
	tagnumber INTEGER DEFAULT NULL, -- unused, moved to ids table
	system_serial VARCHAR(128) DEFAULT NULL,
	location VARCHAR(128) DEFAULT NULL,
	is_broken BOOLEAN DEFAULT NULL,
	disk_removed BOOLEAN DEFAULT NULL,
	department_name VARCHAR(64) DEFAULT NULL,
	ad_domain VARCHAR(64) DEFAULT NULL,
	note VARCHAR(512) DEFAULT NULL,
	client_status VARCHAR(24) DEFAULT NULL,
	building VARCHAR(64) DEFAULT NULL,
	room VARCHAR(64) DEFAULT NULL,
	property_custodian VARCHAR(64) DEFAULT NULL,
	acquired_date TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	retired_date TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	transaction_uuid UUID DEFAULT NULL,
	bulk_update BOOLEAN DEFAULT FALSE,

	CONSTRAINT locations_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL,
	CONSTRAINT locations_department_name_fkey
		FOREIGN KEY (department_name)
			REFERENCES static_department_info(department_name)
		ON UPDATE CASCADE 
		ON DELETE RESTRICT,
	CONSTRAINT locations_ad_domain_fkey
		FOREIGN KEY (ad_domain)
			REFERENCES static_ad_domains(domain_name)
		ON UPDATE CASCADE 
		ON DELETE RESTRICT,
	CONSTRAINT locations_client_status_fkey
		FOREIGN KEY (client_status)
			REFERENCES static_client_statuses(status_name)
		ON UPDATE CASCADE 
		ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS locations_log (
	id SERIAL PRIMARY KEY,
	time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
	client_uuid UUID NOT NULL,
	tagnumber INTEGER DEFAULT NULL, -- unused, moved to ids table
	system_serial VARCHAR(128) DEFAULT NULL,
	location VARCHAR(128) DEFAULT NULL,
	is_broken BOOLEAN DEFAULT NULL,
	disk_removed BOOLEAN DEFAULT NULL,
	department_name VARCHAR(64) DEFAULT NULL,
	ad_domain VARCHAR(64) DEFAULT NULL,
	note VARCHAR(512) DEFAULT NULL,
	client_status VARCHAR(24) DEFAULT NULL,
	building VARCHAR(64) DEFAULT NULL,
	room VARCHAR(64) DEFAULT NULL,
	property_custodian VARCHAR(64) DEFAULT NULL,
	acquired_date TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	retired_date TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	transaction_uuid UUID DEFAULT NULL,
	bulk_update BOOLEAN DEFAULT FALSE,

	CONSTRAINT locations_log_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL,
	CONSTRAINT locations_log_department_name_fkey
		FOREIGN KEY (department_name)
			REFERENCES static_department_info(department_name)
		ON UPDATE CASCADE 
		ON DELETE RESTRICT,
	CONSTRAINT locations_log_ad_domain_fkey
		FOREIGN KEY (ad_domain)
			REFERENCES static_ad_domains(domain_name)
		ON UPDATE CASCADE 
		ON DELETE RESTRICT,
	CONSTRAINT locations_log_client_status_fkey
		FOREIGN KEY (client_status)
			REFERENCES static_client_statuses(status_name)
		ON UPDATE CASCADE 
		ON DELETE RESTRICT
);


DROP TABLE IF EXISTS static_disk_stats;
CREATE TABLE IF NOT EXISTS static_disk_stats (
	disk_model VARCHAR(36) UNIQUE NOT NULL,
	disk_capacity SMALLINT DEFAULT NULL,
	disk_write_speed SMALLINT DEFAULT NULL,
	disk_read_speed SMALLINT DEFAULT NULL,
	disk_mtbf INTEGER DEFAULT NULL,
	max_kbw BIGINT DEFAULT NULL,
	disk_tbr SMALLINT DEFAULT NULL,
	min_temp SMALLINT DEFAULT NULL,
	max_temp SMALLINT DEFAULT NULL,
	disk_interface VARCHAR(4) DEFAULT NULL,
	disk_type VARCHAR(4) DEFAULT NULL,
	spinning BOOLEAN DEFAULT NULL,
	spin_speed SMALLINT DEFAULT NULL,
	power_cycles INTEGER DEFAULT NULL,
	pcie_gen SMALLINT DEFAULT NULL,
	pcie_lanes SMALLINT DEFAULT NULL
);

INSERT INTO static_disk_stats
	(
		disk_model,
		disk_capacity,
		disk_write_speed,
		disk_read_speed,
		disk_mtbf,
		max_kbw,
		disk_tbr,
		min_temp,
		max_temp,
		disk_interface,
		disk_type,
		spinning,
		spin_speed,
		power_cycles,
		pcie_gen,
		pcie_lanes
	)
	VALUES 
		('PM9C1b Samsung 1024GB', 1024, 5600, 6000, 1500000, NULL, NULL, 0, 70, 'm.2', 'nvme', FALSE, NULL, NULL, 4, 4),
		('LITEON CV8-8E128-11 SATA 128GB', 128, 550, 380, 1500000, 146, NULL, 0, 70, 'm.2', 'nvme', FALSE, NULL, 50000, NULL, NULL),
		('MTFDHBA256TCK-1AS1AABHA', NULL, 3000, 1600, 2000000, 75, NULL, NULL, 82, 'm.2', 'nvme', FALSE, NULL, NULL, NULL, NULL),
		('SSDPEMKF256G8 NVMe INTEL 256GB', 256, 3210, 1315, 1600000, 144, NULL, 0, 70, 'm.2', 'nvme', FALSE, NULL, NULL, NULL, NULL),
		('ST500LM034-2GH17A', NULL, 160, 160, NULL, 55, 55, 0, 60, 'sata', 'hdd', TRUE, 200, 600000, NULL, NULL),
		('TOSHIBA MQ01ACF050', NULL, NULL, NULL, 600000, 125, 125, 5, 55,'sata','hdd', TRUE, 7200, NULL, NULL, NULL),
		('WDC PC SN520 SDAPNUW-256G-1006', 256, '1300','1700','1752000','200',NULL,'0','70','m.2','nvme', FALSE,NULL,NULL, NULL, NULL),
		('LITEON CV3-8D512-11 SATA 512GB', 512, '490','540','1500000','250',NULL,NULL,NULL,'m.2','ssd', FALSE,NULL,NULL, NULL, NULL),
		('TOSHIBA KSG60ZMV256G M.2 2280 256GB',256, '340','550','1500000',NULL,NULL,'0','80','m.2','ssd', FALSE,NULL,NULL, NULL, NULL),
		('TOSHIBA THNSNK256GVN8 M.2 2280 256GB', 256, 388, 545, 1500000, 150, NULL, 0, 70, 'm.2', 'nvme', FALSE, NULL, NULL, NULL, NULL),
		('PC SN740 NVMe WD 512GB', 512, '4000','5000','1750000','300',NULL,'0','85','m.2','nvme', FALSE,NULL,'3000', NULL, NULL),
		('SK hynix SC308 SATA 256GB', 256, 130,540,1500000,75,NULL,0,70,'m.2','ssd', FALSE,NULL,NULL, NULL, NULL),
		('ST500LM000-1EJ162', NULL, 100, 100, NULL, 125, 125, 0, 60, 'sata', 'hdd', TRUE, 5400, 25000, NULL, NULL),
		('ST500DM002-1SB10A', NULL, 100, 100, NULL, 125, 125, 0, 60, 'sata', 'hdd', TRUE, 5400, 25000, NULL, NULL),
		('SanDisk SSD PLUS 1000GB', 1000, 350, 535, 26280, 100, NULL, NULL, NULL, 'sata', 'ssd', FALSE, NULL, NULL, NULL, NULL),
		('WDC WD5000LPLX-75ZNTT1', NULL, NULL, NULL, 43800, 125, 125, 0, 60, 'sata', 'hdd', TRUE, 7200, NULL, NULL, NULL),
		('PM991a NVMe Samsung 512GB', 512, 1200, 2200, 1500000, NULL, NULL, 0, 70, 'm.2', 'nvme', FALSE, NULL, NULL, 3, 4)
	;

UPDATE static_disk_stats 
SET max_kbw = max_kbw << 30 
WHERE max_kbw IS NOT NULL;


CREATE TABLE IF NOT EXISTS static_battery_stats (
	battery_model VARCHAR(24) PRIMARY KEY,
	battery_charge_cycles SMALLINT DEFAULT NULL
);

INSERT INTO static_battery_stats
	(
		battery_model,
		battery_charge_cycles
	)
	VALUES 
		('RE03045XL', 300), -- RE03XL --
		('DELL VN3N047', 300),
		('DELL N2K6205', 300),
		('DELL 1VX1H93', 300),
		('DELL W7NKD85', 300),
		('DELL PGFX464', 300),
		('DELL PGFX484', 300),
		('DELL 4M1JN11', 300),
		('X906972', 300),
		('M1009169', 300),
		('X910528', 300)
	ON CONFLICT (battery_model) DO UPDATE SET 
		battery_charge_cycles = EXCLUDED.battery_charge_cycles
;


CREATE TABLE IF NOT EXISTS static_bios_stats (
	system_model VARCHAR(64) PRIMARY KEY,
	bios_version VARCHAR(24) DEFAULT NULL
);

WITH most_recent_firmware_data AS (
	SELECT client_uuid, bios_version FROM (
		SELECT 
			ROW_NUMBER() OVER (PARTITION BY historical_firmware_data.client_uuid ORDER BY time DESC) AS "row_nums", 
			time, 
			historical_firmware_data.client_uuid, 
			historical_firmware_data.bios_version 
		FROM 
			historical_firmware_data 
		WHERE 
			historical_firmware_data.bios_version IS NOT NULL 
	) t1 WHERE t1.row_nums = 1
)
INSERT INTO static_bios_stats (
	system_model,
	bios_version
)
	SELECT 
		t1.system_model, 
		t1.bios_version
	FROM (
		SELECT hardware_data.system_model, most_recent_firmware_data.bios_version, row_number() OVER (PARTITION BY hardware_data.system_model ORDER BY string_to_array(REGEXP_REPLACE(most_recent_firmware_data.bios_version, '[A-Za-z\-\s]', '', 'g'), '.')::int[] DESC) AS row_num
		FROM ids
		LEFT JOIN hardware_data ON ids.uuid = hardware_data.client_uuid
		LEFT JOIN most_recent_firmware_data ON most_recent_firmware_data.client_uuid = hardware_data.client_uuid 
		WHERE 
			hardware_data.system_model IS NOT NULL
			AND most_recent_firmware_data.bios_version IS NOT NULL
		GROUP BY hardware_data.system_model, most_recent_firmware_data.bios_version
		ORDER BY hardware_data.system_model) AS t1
	WHERE 
			t1.row_num = 1
ON CONFLICT (system_model) DO UPDATE SET 
	bios_version = EXCLUDED.bios_version
;

-- VALUES
-- 	('HP ProBook 450 G6', 'R71 Ver. 01.33.00'),
-- 	('Dell Pro Slim Plus QBS1250', '1.6.2'),
-- 	('Latitude 7400', '1.43.0'),
-- 	('OptiPlex 7000', '1.40.0'),
-- 	('Latitude 7420', '1.50.1'),
-- 	('Latitude 3500', '1.36.0'),
-- 	('Latitude 3560', 'A19'),
-- 	('Latitude 3590', '1.26.0'),
-- 	('Latitude 7430', '1.29.0'),
-- 	('Latitude 7490', '1.41.0'),
-- 	('Latitude 7480', '1.40.0'),
-- 	('Latitude E7470', '1.36.3'),
-- 	('OptiPlex 9010 AIO', 'A25'),
-- 	('Latitude E6430', 'A24'),
-- 	('OptiPlex 790', 'A22'),
-- 	('OptiPlex 780', 'A15'),
-- 	('OptiPlex 7460 AIO', '1.35.0'),
-- 	('Latitude 5590', '1.38.0'),
-- 	('XPS 15 9560', '1.24.0'),
-- 	('Latitude 5480', '1.39.0'),
-- 	('Latitude 5289', '1.35.0'),
-- 	('Surface Book', '92.3748.768'),
-- 	('Aspire T3-710', 'R01-B1'),
-- 	('Surface Pro', NULL),
-- 	('Surface Pro 4', '109.3748.768'),
-- 	('OptiPlex 5080', '1.28.1'),
-- 	('OptiPlex 7040', '1.24.0'),
-- 	('OptiPlex 7050', '1.27.0'),
-- 	('OptiPlex 5070', '1.31.1'),
-- 	('OptiPlex 7010', 'A29'),
-- 	('OptiPlex 7780', '1.36.1')




CREATE TABLE IF NOT EXISTS client_health (
	time TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	client_uuid UUID PRIMARY KEY,
	os_name VARCHAR(128) DEFAULT NULL,
	os_installed BOOLEAN DEFAULT NULL,
	disk_free_space_kb BIGINT DEFAULT NULL,
	disk_health_pcnt NUMERIC(6,3) DEFAULT NULL, 
	battery_health_pcnt NUMERIC(6,3) DEFAULT NULL, 
	avg_erase_time SMALLINT DEFAULT NULL, 
	avg_clone_time SMALLINT DEFAULT NULL, 
	last_clone_job_time TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	last_erase_job_time TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	total_jobs_completed SMALLINT DEFAULT NULL,
	last_hardware_check TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	transaction_uuid UUID DEFAULT NULL,
	updated_from_windows BOOLEAN DEFAULT FALSE NOT NULL,

	CONSTRAINT client_health_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS live_os_data (
	client_uuid UUID PRIMARY KEY,
	system_uptime INT DEFAULT NULL,
	client_app_uptime INT DEFAULT NULL,
	last_heard TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	kernel_updated BOOLEAN DEFAULT NULL,
	memory_usage_kb BIGINT DEFAULT NULL,
	cpu_usage DECIMAL(6, 2) DEFAULT NULL,
	network_usage INT DEFAULT NULL,
	link_speed INT DEFAULT NULL,

	CONSTRAINT live_os_data_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL
);

-- INSERT INTO live_os_data (
-- 	client_uuid,
-- 	system_uptime,
-- 	client_app_uptime,
-- 	last_heard,
-- 	kernel_updated,
-- 	memory_usage_kb,
-- 	cpu_usage,
-- 	network_usage,
-- 	link_speed
-- ) SELECT 
-- 	client_uuid,
-- 	system_uptime,
-- 	client_app_uptime,
-- 	last_heard,
-- 	kernel_updated,
-- 	memory_usage_kb,
-- 	cpu_usage,
-- 	network_usage,
-- 	link_speed
-- 	FROM job_queue
-- 	ORDER BY last_heard DESC NULLS LAST;

CREATE TABLE IF NOT EXISTS job_queue (
	client_uuid UUID PRIMARY KEY,
	tagnumber INTEGER DEFAULT NULL, -- should be unused, need to check
	job_queued BOOLEAN DEFAULT FALSE,
	job_name VARCHAR(64) DEFAULT NULL,
	job_queued_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	job_active BOOLEAN DEFAULT FALSE,
	clone_mode VARCHAR(24) DEFAULT NULL,
	erase_mode VARCHAR(24) DEFAULT NULL,
	last_job_time TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	last_heard TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	job_status VARCHAR(128) DEFAULT NULL,
	kernel_updated BOOLEAN DEFAULT NULL,
	battery_charge_pcnt SMALLINT DEFAULT NULL,
	battery_status VARCHAR(20) DEFAULT NULL,
	client_app_uptime INT DEFAULT NULL,
	system_uptime INT DEFAULT NULL,
	disk_temp SMALLINT DEFAULT NULL,
	max_disk_temp SMALLINT DEFAULT NULL,
	watts_now SMALLINT DEFAULT NULL,
	network_speed SMALLINT DEFAULT NULL,
	memory_usage_kb BIGINT DEFAULT NULL,
	memory_capacity_kb BIGINT DEFAULT NULL,
	cpu_usage DECIMAL(6, 2) DEFAULT NULL,
	cpu_mhz INTEGER DEFAULT NULL,
	cpu_temp DECIMAL(6, 2) DEFAULT NULL,
	network_usage INT DEFAULT NULL,
	link_speed INT DEFAULT NULL,

	CONSTRAINT job_queue_job_name_fkey 
		FOREIGN KEY (job_name) 
			REFERENCES static_job_names(job_name) 
		ON UPDATE CASCADE 
		ON DELETE RESTRICT,
	CONSTRAINT job_queue_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL
);

DROP table IF EXISTS logins;
CREATE TABLE IF NOT EXISTS logins (
	username VARCHAR(64) UNIQUE NOT NULL,
	password VARCHAR(60) NOT NULL,
	email VARCHAR(64) DEFAULT NULL,
	first_name VARCHAR(36) DEFAULT NULL,
	last_name VARCHAR(36) DEFAULT NULL,
	common_name VARCHAR(72) NOT NULL,
	role VARCHAR(16) DEFAULT NULL,
	is_admin BOOLEAN NOT NULL DEFAULT FALSE,
	enabled BOOLEAN NOT NULL DEFAULT TRUE,
	two_factor_code VARCHAR(64) DEFAULT NULL
);


CREATE TABLE IF NOT EXISTS hardware_data (
	client_uuid UUID PRIMARY KEY,
	tagnumber INTEGER DEFAULT NULL, -- unused, check if queries still rely on it
	system_serial VARCHAR(128) DEFAULT NULL,
	system_uuid VARCHAR(64) DEFAULT NULL,
	ethernet_mac VARCHAR(17) DEFAULT NULL,
	wifi_mac VARCHAR(17) DEFAULT NULL,
	system_manufacturer VARCHAR(24) DEFAULT NULL,
	system_model VARCHAR(64) DEFAULT NULL,
	system_sku VARCHAR(20) DEFAULT NULL,
	chassis_type VARCHAR(16) DEFAULT NULL,
	device_type VARCHAR(64) DEFAULT NULL,
	tpm_version VARCHAR(24) DEFAULT NULL,
	cpu_manufacturer VARCHAR(20) DEFAULT NULL,
	cpu_model VARCHAR(46) DEFAULT NULL,
	cpu_maxspeed SMALLINT DEFAULT NULL,
	cpu_cores SMALLINT DEFAULT NULL,
	cpu_threads SMALLINT DEFAULT NULL,
	motherboard_manufacturer VARCHAR(24) DEFAULT NULL,
	motherboard_serial VARCHAR(24) DEFAULT NULL,
	time TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	transaction_uuid UUID DEFAULT NULL,
	updated_from_windows BOOLEAN DEFAULT FALSE NOT NULL,

	CONSTRAINT hardware_data_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL,
	CONSTRAINT hardware_data_device_type_fkey
		FOREIGN KEY (device_type)
			REFERENCES static_device_types(device_type)
		ON UPDATE CASCADE
		ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS static_device_types (
	device_type VARCHAR(64) PRIMARY KEY,
	device_type_formatted VARCHAR(64) DEFAULT NULL,
	device_meta_category VARCHAR(64) DEFAULT NULL,
	sort_order SMALLINT DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS bitlocker (
	tagnumber INTEGER PRIMARY KEY,
	identifier VARCHAR(128) NOT NULL,
	recovery_key VARCHAR(128) NOT NULL
);

CREATE TABLE IF NOT EXISTS static_info_tags (
	tag_name VARCHAR(128) PRIMARY KEY,
	tag_readable VARCHAR(128) NOT NULL,
	owner VARCHAR(64) NOT NULL,
	department VARCHAR(128) NOT NULL
);

-- CREATE TABLE IF NOT EXISTS tags (
--     tagnumber VARCHAR(128) NOT NULL,
--     tag VARCHAR(128) NOT NULL
-- );

CREATE TABLE IF NOT EXISTS client_images (
	uuid VARCHAR(128) PRIMARY KEY,
	time TIMESTAMP WITH TIME ZONE NOT NULL,
	client_uuid UUID DEFAULT NULL,
	tagnumber INTEGER NOT NULL, -- unused, check if queries still rely on it
	filename VARCHAR(128) DEFAULT NULL,
	filepath TEXT DEFAULT NULL,
	thumbnail_filename TEXT DEFAULT NULL,
	filesize INTEGER DEFAULT NULL,
	sha256_hash BYTEA DEFAULT NULL,
	mime_type VARCHAR(24) DEFAULT NULL,
	exif_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	resolution_x INTEGER DEFAULT NULL,
	resolution_y INTEGER DEFAULT NULL,
	note VARCHAR(256) DEFAULT NULL,
	hidden BOOLEAN DEFAULT FALSE NOT NULL,
	pinned BOOLEAN DEFAULT FALSE NOT NULL,

	UNIQUE (tagnumber, sha256_hash, hidden),

	CONSTRAINT client_images_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS static_organizations (
	organization_name VARCHAR(64) PRIMARY KEY,
	organization_name_formatted VARCHAR(64) NOT NULL,
	organization_sort_order SMALLINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS static_department_info (
	department_name VARCHAR(64) PRIMARY KEY,
	department_name_formatted VARCHAR(64) NOT NULL,
	department_sort_order  SMALLINT NOT NULL DEFAULT 0,
	department_owner VARCHAR(64) DEFAULT NULL,
	organization_name VARCHAR(64) DEFAULT NULL,

	CONSTRAINT static_department_info_organization_name_fkey
		FOREIGN KEY (organization_name)
			REFERENCES static_organizations(organization_name)
		ON UPDATE CASCADE 
		ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS static_job_names (
	job_name VARCHAR(24) PRIMARY KEY,
	job_name_readable VARCHAR(24) DEFAULT NULL,
	job_sort_order SMALLINT DEFAULT NULL,
	job_hidden BOOLEAN DEFAULT FALSE
);

INSERT INTO 
	static_job_names (job_name, job_name_readable, job_sort_order, job_hidden)
VALUES 
	('update', 'Update Client App', 20, FALSE),
	('findmy', 'Play Sound', 30, FALSE),
	('hpEraseAndClone', 'Erase and Clone', 40, TRUE),
	('generic-erase+clone', 'Erase and Clone (manual)', 41, TRUE),
	('hpCloneOnly', 'Clone Only', 50, FALSE),
	('generic-clone', 'Clone Only (manual)', 51, TRUE),
	('nvmeErase', 'Erase Only', 60, FALSE),
	('generic-erase', 'Erase Only (manual)', 61, TRUE),
	('nvmeVerify', 'Verify Erase', 70, TRUE),
	('shutdown', 'Shutdown', 80, FALSE),
	('cancel', 'Cancel/Clear Job(s)', 95, FALSE)
	ON CONFLICT (job_name) 
	DO UPDATE SET 
		job_name_readable = EXCLUDED.job_name_readable, 
		job_sort_order = EXCLUDED.job_sort_order, 
		job_hidden = EXCLUDED.job_hidden
	;

CREATE TABLE IF NOT EXISTS static_ad_domains (
	domain_name VARCHAR(64) PRIMARY KEY,
	domain_name_formatted VARCHAR(64) DEFAULT NULL,
	domain_sort_order SMALLINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS static_image_names (
	image_name VARCHAR(36) PRIMARY KEY,
	image_os_author VARCHAR(24) DEFAULT NULL,
	image_version VARCHAR(24) DEFAULT NULL,
	image_platform_vendor VARCHAR(24) DEFAULT NULL,
	system_model VARCHAR(36) DEFAULT NULL,
	image_name_readable VARCHAR(36) DEFAULT NULL,
	last_updated TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS notes (
	id SERIAL PRIMARY KEY,
	time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
	note_type VARCHAR(64) DEFAULT NULL,
	note TEXT DEFAULT NULL,
	todo TEXT DEFAULT NULL,
	projects TEXT DEFAULT NULL,
	misc TEXT DEFAULT NULL,
	bugs TEXT DEFAULT NULL,

	CONSTRAINT notes_note_type_fkey
		FOREIGN KEY (note_type)
			REFERENCES static_note_info(note_type)
		ON UPDATE CASCADE 
		ON DELETE RESTRICT
);


CREATE TABLE IF NOT EXISTS static_note_info (
	note_type VARCHAR(64) PRIMARY KEY,
	note_type_readable VARCHAR(64) NOT NULL,
	sort_order SMALLINT DEFAULT NULL
);

INSERT INTO static_note_info (note_type, note_type_readable, sort_order) VALUES 
	('general', 'General Notes', 0),
	('todo', 'Short-Term', 10),
	('projects', 'Projects', 20),
	('misc', 'Misc. Notes', 30),
	('bugs', 'Software Bugs 🐛', 40)
	ON CONFLICT (note_type) DO UPDATE SET
		note_type_readable = EXCLUDED.note_type_readable,
		sort_order = EXCLUDED.sort_order
;


CREATE TABLE IF NOT EXISTS checkout_log (
	transaction_uuid UUID PRIMARY KEY,
	client_uuid UUID NOT NULL,
	time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
	tagnumber INTEGER DEFAULT NULL, -- unused
	customer_name VARCHAR(48) DEFAULT NULL,
	checkout_bool BOOLEAN DEFAULT FALSE,
	checkout_date TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	return_date TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	checkout_group VARCHAR(48) DEFAULT NULL,
	note VARCHAR(512) DEFAULT NULL,

	CONSTRAINT checkout_log_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL,
	CONSTRAINT checkout_log_valid_tag
		CHECK (tagnumber > 100000 AND tagnumber < 999999)
);


-- DROP TABLE IF EXISTS static_emojis;
-- CREATE TABLE IF NOT EXISTS static_emojis (
--     keyword VARCHAR(64) PRIMARY KEY,
--     regex VARCHAR(64) DEFAULT NULL,
--     replacement VARCHAR(64) DEFAULT NULL,
--     text_bool BOOLEAN DEFAULT NULL,
--     case_sensitive_bool BOOLEAN DEFAULT NULL
-- );

-- INSERT INTO static_emojis (keyword, regex, replacement, text_bool, case_sensitive_bool) VALUES 
--   (':)', '\:\)', '😀', NULL, NULL),
--   (':D', '\:D\)', '😁', NULL, TRUE),
--   (';)', '\;\)', '😉', NULL, NULL),
--   (':P', '\:P', '😋', NULL, NULL),
--   (':|', '\:\|', '😑', NULL, NULL),
--   (':0', '\:0', '😲', NULL, NULL),
--   (':O', '\:O', '😲', NULL, NULL),
--   (':(', '\:\(', '😞', NULL, NULL),
--   (':<', '\:\<', '😡', NULL, NULL),
--   (':\', '\:\\', '😕', NULL, NULL),
--   (';(', '\;\(', '😢', NULL, NULL),
--   ('check', '\:check', '✅', TRUE, TRUE),
--   ('done', '\:done', '✅', TRUE, TRUE),
--   ('x', '\:x', '❌', TRUE, NULL),
--   ('cancel', '\:cancel', '🚫', TRUE, TRUE),
--   ('working', '\:working', '⌛', TRUE, TRUE),
--   ('waiting', '\:waiting', '⌛', TRUE, TRUE),
--   ('inprogress', '\:inprogress', '⌛', TRUE, TRUE),
--   ('shurg', '\:shrug', '🤷', TRUE, TRUE),
--   ('clock', '\:clock', '🕓', TRUE, TRUE),
--   ('warning', '\:warning', '⚠️', TRUE, TRUE),
--   ('arrow', '\:arrow', '⏩', TRUE, TRUE),
--   ('bug', '\:bug', '🐛', TRUE, TRUE),
--   ('poop', '\:poop', '💩', TRUE, TRUE),
--   ('star', '\:star', '⭐', TRUE, TRUE),
--   ('heart', '\:heart', '❤️', TRUE, TRUE),
--   ('love', '\:love', '❤️', TRUE, TRUE),
--   ('fire', '\:fire', '🔥', TRUE, TRUE),
--   ('like', '\:like', '👍', TRUE, TRUE),
--   ('dislike', '\:dislike', '👎', TRUE, TRUE),
--   ('info', '\:info', 'ℹ️', TRUE, TRUE),
--   ('pin', '\:pin', '📌', TRUE, TRUE),
--   ('clap', '\:clap', '👏', TRUE, TRUE),
--   ('celebrate', '\:celebrate', '🥳', TRUE, TRUE),
--   ('hmm', '\:hmm', '🤔', TRUE, TRUE),
--   ('alert', '\:alert', '🚨', TRUE, TRUE),
--   ('mindblown', '\:mindblown', '🤯', TRUE, TRUE),
--   ('shock', '\:shock', '⚡', TRUE, TRUE),
--   ('wow', '\:wow', '😲', TRUE, TRUE),
--   ('eyes', '\:eyes', '👀', TRUE, TRUE),
--   ('looking', '\:looking', '👀', TRUE, TRUE)
-- ;

CREATE TABLE IF NOT EXISTS static_client_statuses (
	status_name VARCHAR(24) PRIMARY KEY,
	status_formatted VARCHAR(36) DEFAULT NULL,
	sort_order SMALLINT DEFAULT NULL,
	status_type VARCHAR(24) DEFAULT NULL
);

INSERT INTO 
	static_client_statuses (status_name, status_formatted, sort_order, status_type) 
VALUES
	('in-use', 'In Use', 10, 'Availability'),
	('needs-imaging', 'Needs Erasing/Imaging', 20, 'Preparation'),
	('needs-to-join-domain', 'Needs to Join Domain', 25, 'Preparation'),
	('available', 'Available For Use/Checkout', 30, 'Availability'),
	('reserved-for-checkout', 'Reserved for Checkout', 40, 'Availability'),
	('checked-out', 'Checked Out', 50, 'Availability'),
	('storage', 'In Storage', 60, 'Availability'),
	('needs-repair', 'Needs Repair', 70, 'Maintenance'),
	('pre-property', 'Getting Prepared for Property', 75, 'Preparation'),
	('retired', 'Retired', 80, 'Possession'),
	('lost', 'Lost/Stolen', 90, 'Possession'),
	('other', 'Other', 100, 'Other')
ON CONFLICT (status_name) DO UPDATE SET 
	status_formatted = EXCLUDED.status_formatted, 
	sort_order = EXCLUDED.sort_order, 
	status_type = EXCLUDED.status_type
;

CREATE TABLE IF NOT EXISTS static_building_info (
	building_number VARCHAR(8) PRIMARY KEY,
	building_name VARCHAR(64) NOT NULL,
	building_name_formatted VARCHAR(64) NOT NULL,
	building_sort_order SMALLINT NOT NULL DEFAULT 0
);



CREATE TABLE IF NOT EXISTS static_os_info (
	os_name VARCHAR(128) PRIMARY KEY,
	os_vendor VARCHAR(64) DEFAULT NULL,
	os_platform VARCHAR(64) DEFAULT NULL,
	os_architecture VARCHAR(16) DEFAULT NULL,
	os_version VARCHAR(64) DEFAULT NULL,
	windows_display_version VARCHAR(4) DEFAULT NULL,
	windows_build_number INTEGER DEFAULT NULL,
	windows_ubr INTEGER DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS os_info (
	client_uuid UUID PRIMARY KEY,
	transaction_uuid UUID NOT NULL,
	time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
	os_install_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
	os_vendor VARCHAR(64) DEFAULT NULL,
	os_platform VARCHAR(64) DEFAULT NULL,
	os_architecture VARCHAR(16) DEFAULT NULL,
	os_name VARCHAR(128) NOT NULL,
	os_version VARCHAR(64) DEFAULT NULL,
	windows_display_version VARCHAR(4) DEFAULT NULL,
	windows_build_number INTEGER DEFAULT NULL,
	windows_ubr INTEGER DEFAULT NULL,
	is_disk_encrypted BOOLEAN DEFAULT NULL,
	admin_users TEXT[] DEFAULT NULL,
	computer_name VARCHAR(128) DEFAULT NULL,
	ad_domain VARCHAR(64) DEFAULT NULL,
	ad_computer_name VARCHAR(128) DEFAULT NULL,
	ad_distinguished_name VARCHAR(512) DEFAULT NULL,
	is_intune_joined BOOLEAN DEFAULT NULL,
	secure_boot_enabled BOOLEAN DEFAULT NULL,
	installed_apps TEXT[] DEFAULT NULL,
	updated_from_windows BOOLEAN DEFAULT FALSE NOT NULL,

	CONSTRAINT os_info_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS historical_firmware_data (
	transaction_uuid UUID PRIMARY KEY,
	client_uuid UUID NOT NULL,
	time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
	bios_version VARCHAR(24) DEFAULT NULL,
	bios_release_date TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	bios_firmware VARCHAR(8) DEFAULT NULL,
	has_2023_ca BOOLEAN DEFAULT NULL,
	updated_from_windows BOOLEAN DEFAULT FALSE NOT NULL,

	CONSTRAINT historical_firmware_data_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_historical_firmware_data_time ON historical_firmware_data (time DESC NULLS LAST);

CREATE TABLE IF NOT EXISTS historical_disk_data (
	time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL, 
	transaction_uuid UUID DEFAULT uuidv7() PRIMARY KEY, 
	updated_from_windows BOOLEAN NOT NULL, 
	client_uuid UUID NOT NULL, 
	disk_model VARCHAR(36) DEFAULT NULL, 
	disk_type VARCHAR(4) DEFAULT NULL, 
	disk_size_kb BIGINT DEFAULT NULL, 
	disk_serial VARCHAR(128) DEFAULT NULL, 
	disk_firmware_version VARCHAR(128) DEFAULT NULL,
	disk_reads_kb BIGINT DEFAULT NULL, 
	disk_writes_kb BIGINT DEFAULT NULL, 
	disk_power_cycles INTEGER DEFAULT NULL, 
	disk_power_on_hours INTEGER DEFAULT NULL, 
	disk_error_count INTEGER DEFAULT NULL, 
	disk_errors TEXT[] DEFAULT NULL, 

	CONSTRAINT historical_disk_data_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_historical_disk_data_time ON historical_disk_data (time DESC NULLS LAST);

-- INSERT INTO historical_disk_data (
-- 	time, 
-- 	transaction_uuid, 
-- 	updated_from_windows, 
-- 	client_uuid, 
-- 	disk_model, 
-- 	disk_type,
-- 	disk_size_kb, 
-- 	disk_serial, 
-- 	disk_firmware_version, 
-- 	disk_reads_kb, 
-- 	disk_writes_kb, 
-- 	disk_power_cycles, 
-- 	disk_power_on_hours, 
-- 	disk_error_count
-- ) SELECT 
-- 	time, 
-- 	transaction_uuid, 
-- 	updated_from_windows, 
-- 	client_uuid, 
-- 	disk_model, 
-- 	disk_type,
-- 	disk_size_kb, 
-- 	disk_serial, 
-- 	disk_firmware, 
-- 	disk_reads_kb, 
-- 	disk_writes_kb, 
-- 	disk_power_cycles, 
-- 	disk_power_on_hours, 
-- 	disk_errors
-- 	FROM historical_hardware_data
-- 	WHERE time IS NOT NULL
-- 	ORDER BY time DESC NULLS LAST;

CREATE TABLE IF NOT EXISTS historical_battery_data (
	time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL, 
	transaction_uuid UUID DEFAULT uuidv7() PRIMARY KEY,  
	updated_from_windows BOOLEAN NOT NULL, 
	client_uuid UUID NOT NULL, 
	battery_serial VARCHAR(64) DEFAULT NULL, 
	battery_manufacturer VARCHAR(64) DEFAULT NULL, 
	battery_model VARCHAR(64) DEFAULT NULL, 
	battery_charge_cycles INTEGER DEFAULT NULL, 
	battery_design_capacity INTEGER DEFAULT NULL,
	battery_manufacture_date DATE DEFAULT NULL, 
	battery_current_max_capacity INTEGER DEFAULT NULL, 

	CONSTRAINT historical_battery_data_client_uuid_fkey
		FOREIGN KEY (client_uuid)
			REFERENCES ids(uuid)
		ON UPDATE CASCADE
		ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_historical_battery_data_time ON historical_battery_data (time DESC NULLS LAST);

-- INSERT INTO historical_battery_data (
-- 	time, 
-- 	transaction_uuid, 
-- 	updated_from_windows, 
-- 	client_uuid, 
-- 	battery_serial, 
-- 	battery_manufacturer, 
-- 	battery_model, 
-- 	battery_charge_cycles, 
-- 	battery_design_capacity,
-- 	battery_manufacture_date, 
-- 	battery_current_max_capacity
-- ) SELECT 
-- 	time, 
-- 	transaction_uuid, 
-- 	updated_from_windows, 
-- 	client_uuid, 
-- 	battery_serial, 
-- 	battery_manufacturer, 
-- 	battery_model, 
-- 	battery_charge_cycles, 
-- 	battery_design_capacity,
-- 	battery_manufacture_date, 
-- 	battery_current_max_capacity
-- 	FROM historical_hardware_data
-- 	WHERE time IS NOT NULL
-- 	ORDER BY time DESC NULLS LAST;