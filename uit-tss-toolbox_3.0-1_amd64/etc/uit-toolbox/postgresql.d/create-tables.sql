-- Tables with tagnmbers: clientstats, jobstats, locations, client_health, job_queue, hardware_data, bitlocker, checkout
DROP TABLE IF EXISTS serverstats;
CREATE TABLE serverstats (
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
	uuid VARCHAR(64) UNIQUE NOT NULL,
	tagnumber INTEGER DEFAULT NULL,
	etheraddress VARCHAR(17) DEFAULT NULL,
	date DATE DEFAULT NULL,
	time TIMESTAMP(3) WITH TIME ZONE DEFAULT NULL,
	system_serial VARCHAR(128) DEFAULT NULL,
	disk VARCHAR(8) DEFAULT NULL,
	disk_model VARCHAR(36) DEFAULT NULL,
	disk_type VARCHAR(4) DEFAULT NULL,
	disk_size SMALLINT DEFAULT NULL,
	disk_serial VARCHAR(32) DEFAULT NULL,
	disk_writes DECIMAL(5,2) DEFAULT NULL,
	disk_reads DECIMAL(5,2) DEFAULT NULL,
	disk_power_on_hours INTEGER DEFAULT NULL,
	disk_errors INT DEFAULT NULL,
	disk_power_cycles INTEGER DEFAULT NULL,
	disk_temp SMALLINT DEFAULT NULL,
	disk_firmware VARCHAR(10) DEFAULT NULL,
	battery_model VARCHAR(16) DEFAULT NULL,
	battery_serial VARCHAR(16) DEFAULT NULL,
	battery_health SMALLINT DEFAULT NULL,
	battery_charge_cycles SMALLINT DEFAULT NULL,
	battery_capacity INTEGER DEFAULT NULL,
	battery_manufacturedate DATE DEFAULT NULL,
	cpu_temp SMALLINT DEFAULT NULL,
	bios_version VARCHAR(24) DEFAULT NULL,
	bios_date VARCHAR(12) DEFAULT NULL,
	bios_firmware VARCHAR(8) DEFAULT NULL,
	ram_serial VARCHAR(128) DEFAULT NULL,
	ram_capacity SMALLINT DEFAULT NULL,
	ram_speed SMALLINT DEFAULT NULL,
	cpu_usage DECIMAL(6,2) DEFAULT NULL,
	network_usage DECIMAL(5,2) DEFAULT NULL,
	boot_time DECIMAL(5,2) DEFAULT NULL,
	erase_completed BOOLEAN DEFAULT FALSE,
	erase_mode VARCHAR(24) DEFAULT NULL,
	erase_diskpercent SMALLINT DEFAULT NULL,
	erase_time SMALLINT DEFAULT NULL,
	clone_completed BOOLEAN DEFAULT FALSE,
	clone_image VARCHAR(36) DEFAULT NULL,
	clone_master BOOLEAN DEFAULT FALSE,
	clone_time SMALLINT DEFAULT NULL,
	job_failed BOOLEAN DEFAULT FALSE,
	host_connected BOOLEAN DEFAULT FALSE
);


CREATE TABLE IF NOT EXISTS locations (
	time TIMESTAMP(3) WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP PRIMARY KEY,
	tagnumber INTEGER NOT NULL,
	system_serial VARCHAR(128) DEFAULT NULL,
	location VARCHAR(128) DEFAULT NULL,
	is_broken BOOLEAN DEFAULT NULL,
	disk_removed BOOLEAN DEFAULT NULL,
	department_name VARCHAR(64) REFERENCES static_department_info(department_name) DEFAULT NULL,
	ad_domain VARCHAR(64) REFERENCES static_ad_domains(domain_name) DEFAULT NULL,
	note VARCHAR(512) DEFAULT NULL,
	client_status VARCHAR(24) REFERENCES static_client_statuses(status) DEFAULT NULL,
	building VARCHAR(64) DEFAULT NULL,
	room VARCHAR(64) DEFAULT NULL,
	property_custodian VARCHAR(64) DEFAULT NULL,
	acquired_date TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	retired_date TIMESTAMP WITH TIME ZONE DEFAULT NULL,
	transaction_uuid UUID DEFAULT NULL
);


DROP TABLE IF EXISTS static_disk_stats;
CREATE TABLE IF NOT EXISTS static_disk_stats (
	disk_model VARCHAR(36) UNIQUE NOT NULL,
	disk_capacity SMALLINT DEFAULT NULL,
	disk_write_speed SMALLINT DEFAULT NULL,
	disk_read_speed SMALLINT DEFAULT NULL,
	disk_mtbf INTEGER DEFAULT NULL,
	disk_tbw SMALLINT DEFAULT NULL,
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
		disk_tbw,
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


CREATE TABLE IF NOT EXISTS static_battery_stats (
	battery_model VARCHAR(24) UNIQUE NOT NULL,
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
	ON CONFLICT (battery_model) DO UPDATE SET battery_charge_cycles = EXCLUDED.battery_charge_cycles
	;


CREATE TABLE IF NOT EXISTS static_bios_stats (
	system_model VARCHAR(64) UNIQUE NOT NULL,
	bios_version VARCHAR(24) DEFAULT NULL
);

INSERT INTO static_bios_stats
	(
		system_model,
		bios_version
	)
VALUES
	('HP ProBook 450 G6', 'R71 Ver. 01.33.00'),
	('Dell Pro Slim Plus QBS1250', '1.6.2'),
	('Latitude 7400', '1.41.1'),
	('OptiPlex 7000', '1.31.1'),
	('Latitude 7420', '1.43.1'),
	('Latitude 3500', '1.36.0'),
	('Latitude 3560', 'A19'),
	('Latitude 3590', '1.26.0'),
	('Latitude 7430', '1.29.0'),
	('Latitude 7490', '1.41.0'),
	('Latitude 7480', '1.40.0'),
	('Latitude E7470', '1.36.3'),
	('OptiPlex 9010 AIO', 'A25'),
	('Latitude E6430', 'A24'),
	('OptiPlex 790', 'A22'),
	('OptiPlex 780', 'A15'),
	('OptiPlex 7460 AIO', '1.35.0'),
	('Latitude 5590', '1.38.0'),
	('XPS 15 9560', '1.24.0'),
	('Latitude 5480', '1.39.0'),
	('Latitude 5289', '1.35.0'),
	('Surface Book', '92.3748.768'),
	('Aspire T3-710', 'R01-B1'),
	('Surface Pro', NULL),
	('Surface Pro 4', '109.3748.768'),
	('OptiPlex 5080', '1.28.1'),
	('OptiPlex 7040', '1.24.0'),
	('OptiPlex 7050', '1.27.0'),
	('OptiPlex 5070', '1.31.1'),
	('OptiPlex 7010', 'A29'),
	('OptiPlex 7780', '1.36.1')
ON CONFLICT (system_model) DO UPDATE SET bios_version = EXCLUDED.bios_version
;


CREATE TABLE IF NOT EXISTS client_health (
	time TIMESTAMP(3) WITH TIME ZONE DEFAULT NULL,
	tagnumber INTEGER UNIQUE NOT NULL,
	system_serial VARCHAR(128) DEFAULT NULL,
	tpm_version VARCHAR(24) DEFAULT NULL,
	bios_version VARCHAR(24) DEFAULT NULL,
	bios_updated BOOLEAN DEFAULT NULL,
	os_name VARCHAR(24) DEFAULT NULL,
	os_installed BOOLEAN DEFAULT NULL,
	disk_type VARCHAR(4) DEFAULT NULL, 
	disk_health NUMERIC(6,3) DEFAULT NULL, 
	battery_health NUMERIC(6,3) DEFAULT NULL, 
	avg_erase_time SMALLINT DEFAULT NULL, 
	avg_clone_time SMALLINT DEFAULT NULL, 
	last_imaged_time TIMESTAMP(3) WITH TIME ZONE DEFAULT NULL,
	all_jobs SMALLINT DEFAULT NULL,
	last_hardware_check TIMESTAMP(3) WITH TIME ZONE DEFAULT NULL,
	transaction_uuid UUID DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS job_queue (
	tagnumber INTEGER UNIQUE NOT NULL,
	job_queued VARCHAR(24) DEFAULT NULL,
	job_queued_position SMALLINT DEFAULT NULL,
	job_active BOOLEAN DEFAULT FALSE,
	clone_mode VARCHAR(24) DEFAULT NULL,
	erase_mode VARCHAR(24) DEFAULT NULL,
	last_job_time TIMESTAMP(3) WITH TIME ZONE DEFAULT NULL,
	present TIMESTAMP(3) WITH TIME ZONE DEFAULT NULL,
	present_bool BOOLEAN DEFAULT FALSE,
	status VARCHAR(128) DEFAULT NULL,
	kernel_updated BOOLEAN DEFAULT NULL,
	battery_charge SMALLINT DEFAULT NULL,
	battery_status VARCHAR(20) DEFAULT NULL,
	uptime INT DEFAULT NULL,
	disk_temp SMALLINT DEFAULT NULL,
	max_disk_temp SMALLINT DEFAULT NULL,
	watts_now SMALLINT DEFAULT NULL,
	network_speed SMALLINT DEFAULT NULL,
	memory_usage DECIMAL(6, 2) DEFAULT NULL,
	memory_capacity DECIMAL(6, 2) DEFAULT NULL,
	cpu_usage DECIMAL(6, 2) DEFAULT NULL,
	cpu_temp DECIMAL(6, 2) DEFAULT NULL,
	network_usage INT DEFAULT NULL,
	link_speed INT DEFAULT NULL
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
	enabled BOOLEAN NOT NULL DEFAULT TRUE
);


CREATE TABLE IF NOT EXISTS hardware_data (
	tagnumber INTEGER UNIQUE NOT NULL,
	etheraddress VARCHAR(17) DEFAULT NULL,
	wifi_mac VARCHAR(17) DEFAULT NULL,
	system_manufacturer VARCHAR(24) DEFAULT NULL,
	system_model VARCHAR(64) DEFAULT NULL,
	system_uuid VARCHAR(64) DEFAULT NULL,
	system_sku VARCHAR(20) DEFAULT NULL,
	chassis_type VARCHAR(16) DEFAULT NULL,
	device_type VARCHAR(64) REFERENCES static_device_types(device_type) DEFAULT NULL,
	cpu_manufacturer VARCHAR(20) DEFAULT NULL,
	cpu_model VARCHAR(46) DEFAULT NULL,
	cpu_maxspeed SMALLINT DEFAULT NULL,
	cpu_cores SMALLINT DEFAULT NULL,
	cpu_threads SMALLINT DEFAULT NULL,
	motherboard_manufacturer VARCHAR(24) DEFAULT NULL,
	motherboard_serial VARCHAR(24) DEFAULT NULL,
	time TIMESTAMP(3) WITH TIME ZONE DEFAULT NULL,
	transaction_uuid UUID DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS static_device_types (
	device_type VARCHAR(64) PRIMARY KEY,
	device_type_formatted VARCHAR(64) DEFAULT NULL,
	device_meta_category VARCHAR(64) DEFAULT NULL,
	sort_order SMALLINT DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS bitlocker (
	tagnumber INTEGER UNIQUE NOT NULL,
	identifier VARCHAR(128) NOT NULL,
	recovery_key VARCHAR(128) NOT NULL
);

CREATE TABLE IF NOT EXISTS static_tags (
	tag VARCHAR(128) NOT NULL,
	tag_readable VARCHAR(128) NOT NULL,
	owner VARCHAR(64) NOT NULL,
	department VARCHAR(128) NOT NULL
);

-- CREATE TABLE IF NOT EXISTS tags (
--     tagnumber VARCHAR(128) NOT NULL,
--     tag VARCHAR(128) NOT NULL
-- );

CREATE TABLE IF NOT EXISTS client_images (
	uuid VARCHAR(128) UNIQUE NOT NULL,
	time TIMESTAMP(3) WITH TIME ZONE NOT NULL,
	tagnumber INTEGER NOT NULL, 
	filename VARCHAR(128) DEFAULT NULL,
	filepath TEXT DEFAULT NULL,
	thumbnail_filepath TEXT DEFAULT NULL,
	filesize INTEGER DEFAULT NULL,
	sha256_hash BYTEA DEFAULT NULL,
	mime_type VARCHAR(24) DEFAULT NULL,
	exif_timestamp TIMESTAMP(3) WITH TIME ZONE DEFAULT NULL,
	resolution_x INTEGER DEFAULT NULL,
	resolution_y INTEGER DEFAULT NULL,
	note VARCHAR(256) DEFAULT NULL,
	hidden BOOLEAN DEFAULT FALSE NOT NULL,
	pinned BOOLEAN DEFAULT FALSE NOT NULL,
	UNIQUE (tagnumber, sha256_hash, hidden)
);

-- CREATE OR REPLACE FUNCTION live_images_function
-- CREATE OR REPLACE TRIGGER live_images_trigger AFTER UPDATE OF screenshot ON live_images FOR EACH ROW EXECUTE FUNCTION live_images_function();
CREATE TABLE IF NOT EXISTS live_images (
	tagnumber INTEGER UNIQUE NOT NULL,
	time TIMESTAMP(3) WITH TIME ZONE DEFAULT NULL,
	screenshot TEXT DEFAULT NULL
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
	organization_name VARCHAR(64) REFERENCES static_organizations(organization_name) DEFAULT NULL
);

DROP TABLE IF EXISTS static_job_names;
CREATE TABLE IF NOT EXISTS static_job_names (
	job_name VARCHAR(24) PRIMARY KEY,
	job_name_readable VARCHAR(24) DEFAULT NULL,
	job_sort_order SMALLINT DEFAULT NULL,
	job_hidden BOOLEAN DEFAULT FALSE
);

INSERT INTO 
	static_job_names (job_name, job_name_readable, job_sort_order, job_hidden)
VALUES 
	('update', 'Update', 20, FALSE),
	('findmy', 'Play Sound', 30, FALSE),
	('hpEraseAndClone', 'Erase and Clone', 40, TRUE),
	('generic-erase+clone', 'Erase and Clone (manual)', 41, TRUE),
	('hpCloneOnly', 'Clone Only', 50, FALSE),
	('generic-clone', 'Clone Only (manual)', 51, TRUE),
	('nvmeErase', 'Erase Only', 60, FALSE),
	('generic-erase', 'Erase Only (manual)', 61, TRUE),
	('nvmeVerify', 'Verify Erase', 70, TRUE),
	('data collection', 'Data Collection', 80, TRUE),
	('shutdown', 'Shutdown', 90, TRUE),
	('clean-shutdown', 'Shutdown', 91, FALSE),
	('cancel', 'Cancel/Clear Job(s)', 95, FALSE)
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
    image_platform_model VARCHAR(36) DEFAULT NULL,
    image_name_readable VARCHAR(36) DEFAULT NULL
);

INSERT INTO
    static_image_names (image_name, image_os_author, image_version, image_platform_vendor, image_platform_model, image_name_readable)
VALUES
    ('TechCommons-HP-LaptopsLZ4', 'Microsoft', 'Windows 11', 'HP', 'HP ProBook 450 G6', 'Windows 11'),
    ('TechCommons-Dell-Desktop-Team-Leads', 'Microsoft', 'Windows 11', 'HP', 'Dell Pro Slim Plus QBS1250', 'Windows 11'),
    ('TechCommons-Dell-Laptops', 'Microsoft', 'Windows 11', 'Dell', 'Latitude 7400', 'Windows 11'),
    ('TechCommons-Dell-Desktops', 'Microsoft', 'Windows 11', 'Dell', 'OptiPlex 7000', 'Windows 11'),
    ('TechCommons-Dell-HelpDesk', 'Microsoft', 'Windows 11', 'Dell', 'Latitude 7420', 'Windows 11'),
    ('SHRL-Dell-Desktops', 'Microsoft', 'Windows 11', 'Dell', NULL, 'Windows 11'),
    ('Ubuntu-Desktop', 'Canonical', '24.04.2 LTS', 'Dell', NULL, 'Ubuntu Desktop')
    ON CONFLICT (image_name) DO NOTHING
    ;


CREATE TABLE IF NOT EXISTS notes (
    time TIMESTAMP(3) WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP PRIMARY KEY,
    note_type VARCHAR(64) DEFAULT NULL,
    note TEXT DEFAULT NULL,
    todo TEXT DEFAULT NULL,
    projects TEXT DEFAULT NULL,
    misc TEXT DEFAULT NULL,
    bugs TEXT DEFAULT NULL
);


CREATE TABLE IF NOT EXISTS static_note_info (
	note_type VARCHAR(64) PRIMARY KEY,
	note_type_readable VARCHAR(64) NOT NULL,
	sort_order SMALLINT DEFAULT NULL
);

INSERT INTO static_note_info (note_type, note_type_readable, sort_order) VALUES 
	('todo', 'Short-Term', 10),
	('projects', 'Projects', 20),
	('misc', 'Misc. Notes', 30),
	('bugs', 'Software Bugs 🐛', 40)
	ON CONFLICT (note_type) DO NOTHING
;


CREATE TABLE IF NOT EXISTS checkout_log (
	log_entry_time TIMESTAMP(3) WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP PRIMARY KEY,
	transaction_uuid UUID DEFAULT NULL,
	tagnumber INTEGER NOT NULL,
	customer_name VARCHAR(48) DEFAULT NULL,
	checkout_bool BOOLEAN DEFAULT FALSE,
	checkout_date DATE DEFAULT NULL,
	return_date DATE DEFAULT NULL,
	checkout_group VARCHAR(48) DEFAULT NULL,
	note VARCHAR(512) DEFAULT NULL
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
    status VARCHAR(24) PRIMARY KEY,
    status_formatted VARCHAR(36) DEFAULT NULL,
    sort_order SMALLINT DEFAULT NULL
);

INSERT INTO static_client_statuses (status, status_formatted, sort_order) VALUES
  ('in-use', 'In Use', 10),
  ('available', 'Available', 20),
  ('needs-imaging', 'Needs Imaging', 30),
  ('reserved-for-checkout', 'Reserved for Checkout', 40),
  ('checked-out', 'Checked Out', 50),
  ('storage', 'Storage', 60),
  ('needs-repair', 'Needs Repair', 70),
  ('retired', 'Retired', 80),
  ('lost', 'Lost/Stolen', 90),
  ('other', 'Other', 100) ON CONFLICT (status) DO NOTHING
  ;