-- Tables with tagnmbers: clientstats, jobstats, locations, client_health, remote, system_data, bitlocker, checkout
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
    time TIMESTAMP(3) DEFAULT NULL,
    system_serial VARCHAR(24) DEFAULT NULL,
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
    time TIMESTAMP(3) UNIQUE NOT NULL,
    tagnumber INTEGER DEFAULT NULL,
    system_serial VARCHAR(24) DEFAULT NULL,
    location VARCHAR(128) DEFAULT NULL,
    status BOOLEAN DEFAULT NULL,
    disk_removed BOOLEAN DEFAULT NULL,
    department VARCHAR(24) DEFAULT NULL,
    domain VARCHAR(24) DEFAULT NULL,
    note VARCHAR(512) DEFAULT NULL
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
    (disk_model,
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
    pcie_lanes)
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


DROP TABLE IF EXISTS static_battery_stats;
CREATE TABLE IF NOT EXISTS static_battery_stats (
    battery_model VARCHAR(24) UNIQUE NOT NULL,
    battery_charge_cycles SMALLINT DEFAULT NULL
);

INSERT INTO static_battery_stats
    (battery_model,
    battery_charge_cycles
    )
VALUES 
    ('RE03045XL', 300),
    ('DELL VN3N047', 300),
    ('DELL N2K6205', 300),
    ('DELL 1VX1H93', 300),
    ('DELL W7NKD85', 300),
    ('DELL PGFX464', 300),
    ('DELL PGFX484', 300),
    ('DELL 4M1JN11', 300),
    ('X906972', 300),
    ('M1009169', 300),
    ('X910528', 300);


DROP TABLE IF EXISTS static_bios_stats;
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
    ('HP ProBook 450 G6', 'R71 Ver. 01.32.00'),
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
    ('OptiPlex 7780', '1.36.1');


CREATE TABLE IF NOT EXISTS client_health (
    tagnumber INTEGER UNIQUE NOT NULL,
    system_serial VARCHAR(24) DEFAULT NULL,
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
    last_imaged_time TIMESTAMP(3) DEFAULT NULL,
    all_jobs SMALLINT DEFAULT NULL,
    time TIMESTAMP(3) DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS remote (
    tagnumber INTEGER UNIQUE NOT NULL,
    job_queued VARCHAR(24) DEFAULT NULL,
    job_queued_position SMALLINT DEFAULT NULL,
    job_active BOOLEAN DEFAULT FALSE,
    clone_mode VARCHAR(24) DEFAULT NULL,
    erase_mode VARCHAR(24) DEFAULT NULL,
    last_job_time TIMESTAMP(3) DEFAULT NULL,
    present TIMESTAMP DEFAULT NULL,
    present_bool BOOLEAN DEFAULT FALSE,
    status VARCHAR(128) DEFAULT NULL,
    kernel_updated BOOLEAN DEFAULT NULL,
    battery_charge SMALLINT DEFAULT NULL,
    battery_status VARCHAR(20) DEFAULT NULL,
    uptime INT DEFAULT NULL,
    cpu_temp SMALLINT DEFAULT NULL,
    disk_temp SMALLINT DEFAULT NULL,
    max_disk_temp SMALLINT DEFAULT NULL,
    watts_now SMALLINT DEFAULT NULL,
    network_speed SMALLINT DEFAULT NULL
);


DROP table IF EXISTS logins;
CREATE TABLE IF NOT EXISTS logins (
    username VARCHAR(128) UNIQUE NOT NULL,
    password VARCHAR(128) NOT NULL,
    name VARCHAR(36) NOT NULL
);


CREATE TABLE IF NOT EXISTS system_data (
    tagnumber INTEGER UNIQUE NOT NULL,
    etheraddress VARCHAR(17) DEFAULT NULL,
    wifi_mac VARCHAR(17) DEFAULT NULL,
    system_manufacturer VARCHAR(24) DEFAULT NULL,
    system_model VARCHAR(64) DEFAULT NULL,
    system_uuid VARCHAR(64) DEFAULT NULL,
    system_sku VARCHAR(20) DEFAULT NULL,
    chassis_type VARCHAR(16) DEFAULT NULL,
    cpu_manufacturer VARCHAR(20) DEFAULT NULL,
    cpu_model VARCHAR(46) DEFAULT NULL,
    cpu_maxspeed SMALLINT DEFAULT NULL,
    cpu_cores SMALLINT DEFAULT NULL,
    cpu_threads SMALLINT DEFAULT NULL,
    motherboard_manufacturer VARCHAR(24) DEFAULT NULL,
    motherboard_serial VARCHAR(24) DEFAULT NULL,
    time TIMESTAMP(3) DEFAULT NULL
);


DROP TABLE IF EXISTS bitlocker;
CREATE TABLE IF NOT EXISTS bitlocker (
    tagnumber INTEGER UNIQUE NOT NULL,
    identifier VARCHAR(128) NOT NULL,
    recovery_key VARCHAR(128) NOT NULL
);

INSERT INTO bitlocker
(
    tagnumber,
    identifier,
    recovery_key
) VALUES
('724020', '94A6BC65-EBC1-4BF7-9B6B-45FE14102851', '051139-186978-171864-593648-529529-634975-122133-502480'),
('724021', 'A6E98E82-2A03-4BF6-98C7-DF545A7745D5', '143429-284900-250481-657690-514668-500269-430419-492866'),
('724025', '18741871-6DE8-41CE-AF96-7026F4AF12B5', '313247-612755-597828-287078-399916-220363-401357-318516'),
('727005', '956EAC0D-1E93-4F8A-95EE-96F4B0CE9CC1', '568172-561781-653939-211431-673860-559493-507177-527901'),
('727006', '5FEB2D5C-5A3C-43D5-ABDD-2FF2D7F000E9', '209869-405218-331342-499939-399025-101068-711370-621940'),
('727007', 'E78C562F-A1D7-409E-9D1A-F8F4DAD7CAB3', '453189-289113-568480-026411-025366-143473-390137-104456'),
('727008', 'ABD175B0-F20B-4A04-BEA8-60D69214DDD2', '141570-153043-215072-647757-021956-291082-215149-107910'),
('727009', '6ECFDB34-5334-4452-8F86-E9043FED3C05', '346841-378015-041173-326898-155243-358083-076736-553894'),
('727010', 'E21543FA-CE92-4BDE-AE90-4C713E0ACFEF', '502821-236709-657030-654819-639243-351824-278762-099869'),
('727011', '3536669E-2D87-4F89-AA77-9289B987A97C', '123651-645084-470547-555709-391039-523380-469370-552761'),
('727012', 'C3373E7C-F19A-4AFA-BDE9-320431B1DD80', '655930-154462-345026-620345-020020-002761-693352-541145'),
('727013', '7C23FCCC-8430-405F-BDDE-23B149E1B926', '153736-395956-001419-128854-250800-440308-637593-212146'),
('727014', '0761161E-23A5-461B-822D-6D795B7253DB', '100870-342463-572495-557854-219527-118217-661925-117348'),
('625806', 'CBF86C66-7178-4708-BB33-E9377DA3FED1', '515812-612612-609510-498311-368049-512424-262372-073986')
;


DROP TABLE IF EXISTS static_tags;
CREATE TABLE IF NOT EXISTS static_tags (
    tag VARCHAR(128) NOT NULL,
    tag_readable VARCHAR(128) NOT NULL,
    owner VARCHAR(64) NOT NULL,
    department VARCHAR(128) NOT NULL
);

INSERT INTO static_tags (
    tag,
    tag_readable,
    owner,
    department
) VALUES 
    ('laptop-program', 'Laptop Program', 'Matthew Harvey', 'techComm'),
    ('checked-in', 'Checked In', 'Matthew Harvey', 'techComm'),
    ('checked-out', 'Checked Out', 'Matthew Harvey', 'techComm'),
    ('stolen', 'Stolen/Missing', 'Matthew Harvey', 'techComm'),
    ('ITSC-Computers', 'ITSC Team Leads', 'Kevin Vu', 'techComm'),
    ('tv', 'Televisions', 'Kevin Vu', 'techComm'),
    ('security-cameras', 'Security Cameras', 'Kevin Vu', 'techComm'),
    ('student-workstations', 'Student Workstations', 'Kevin Vu', 'techComm'),
    ('printers', 'Printers', 'Tom Carroll', 'o365'),
    ('uniprint-clients', 'Uniprint Release Stations', 'Tom Carroll', 'o365')
    ;


-- CREATE TABLE IF NOT EXISTS tags (
--     tagnumber VARCHAR(128) NOT NULL,
--     tag VARCHAR(128) NOT NULL
-- );

CREATE TABLE IF NOT EXISTS client_images (
    uuid VARCHAR(64) UNIQUE NOT NULL,
    time TIMESTAMP(3) NOT NULL,
    tagnumber INTEGER NOT NULL, 
    filename VARCHAR(64) DEFAULT NULL,
    filesize DECIMAL(5,2) DEFAULT NULL,
    image TEXT DEFAULT NULL,
    thumbnail TEXT DEFAULT NULL,
    md5_hash VARCHAR(32) DEFAULT NULL,
    mime_type VARCHAR(24) DEFAULT NULL,
    exif_timestamp TIMESTAMP(3) DEFAULT NULL,
    resolution VARCHAR(24) DEFAULT NULL,
    note VARCHAR(256) DEFAULT NULL,
    hidden BOOLEAN DEFAULT FALSE,
    primary_image BOOLEAN DEFAULT FALSE
);

-- CREATE OR REPLACE FUNCTION live_images_function
-- CREATE OR REPLACE TRIGGER live_images_trigger AFTER UPDATE OF screenshot ON live_images FOR EACH ROW EXECUTE FUNCTION live_images_function();
CREATE TABLE IF NOT EXISTS live_images (
    tagnumber INTEGER UNIQUE NOT NULL,
    time TIMESTAMP(3) DEFAULT NULL,
    screenshot TEXT DEFAULT NULL
);

DROP TABLE IF EXISTS static_departments;
CREATE TABLE IF NOT EXISTS static_departments (
  department VARCHAR(128) UNIQUE NOT NULL,
  department_readable VARCHAR(128) NOT NULL,
  owner VARCHAR(64) NOT NULL,
  department_bool BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO static_departments (
  department,
  department_readable,
  owner,
  department_bool
) VALUES
  ('techComm', 'Tech Commons', 'Matthew Harvey', TRUE),
  ('execSupport', 'Exec. Support', 'Kevin Vu', TRUE),
  ('shrl', 'SHRL', 'Alex Tran', TRUE),
  ('pre-property', 'Pre-Property', 'Matthew Harvey', FALSE),
  ('o365', 'Office 365', 'Andy Moon', FALSE),
  ('property', 'Property', 'Unknown', FALSE),
  ('tss-apple', 'Apple Devices', 'Mark Norgan', FALSE),
  ('uit-security', 'UIT Security', 'Kevin Vu', FALSE),
  ('itac', 'ITAC', 'ITAC', FALSE)
;

DROP TABLE IF EXISTS static_job_names;
CREATE TABLE IF NOT EXISTS static_job_names (
  job VARCHAR(24) UNIQUE NOT NULL,
  job_readable VARCHAR(24) DEFAULT NULL,
  job_rank SMALLINT DEFAULT NULL,
  job_html_bool BOOLEAN DEFAULT NULL
);

INSERT INTO 
    static_job_names (job, job_readable, job_rank, job_html_bool)
VALUES 
    ('update', 'Update', 20, TRUE),
    ('findmy', 'Play Sound', 30, TRUE),
    ('hpEraseAndClone', 'Erase and Clone', 40, FALSE),
    ('generic-erase+clone', 'Erase and Clone (manual)', 41, FALSE),
    ('hpCloneOnly', 'Clone Only', 50, TRUE),
    ('generic-clone', 'Clone Only (manual)', 51, FALSE),
    ('nvmeErase', 'Erase Only', 60, TRUE),
    ('generic-erase', 'Erase Only (manual)', 61, FALSE),
    ('nvmeVerify', 'Verify Erase', 70, FALSE),
    ('data collection', 'Data Collection', 80, FALSE),
    ('shutdown', 'Shutdown', 90, FALSE),
    ('clean-shutdown', 'Shutdown', 91, TRUE),
    ('cancel', 'Cancel/Clear Job(s)', 95, TRUE)
    ;


DROP TABLE IF EXISTS static_domains;
CREATE TABLE IF NOT EXISTS static_domains (
  domain VARCHAR(24) UNIQUE NOT NULL,
  domain_readable VARCHAR(36) DEFAULT NULL
);

INSERT INTO 
    static_domains (domain, domain_readable)
VALUES
    ('IT-TSS-CheckOut', 'TSS Laptop Checkout'),
    ('IT-TSS-Teamleads', 'TSS ITSC Team Leads')
    ;


DROP TABLE IF EXISTS static_image_names;
CREATE TABLE IF NOT EXISTS static_image_names (
    image_name VARCHAR(36) UNIQUE NOT NULL,
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
    ;


CREATE TABLE IF NOT EXISTS notes (
    time TIMESTAMP(3) UNIQUE NOT NULL,
    todo TEXT DEFAULT NULL,
    projects TEXT DEFAULT NULL,
    misc TEXT DEFAULT NULL,
    bugs TEXT DEFAULT NULL
);

DROP TABLE IF EXISTS static_notes;
CREATE TABLE IF NOT EXISTS static_notes (
    note VARCHAR(64) UNIQUE NOT NULL,
    note_readable VARCHAR(64) NOT NULL,
    sort_order SMALLINT DEFAULT NULL
);

INSERT INTO static_notes (note, note_readable, sort_order) VALUES 
    ('todo', 'Short-Term', 10),
    ('projects', 'Projects', 20),
    ('misc', 'Misc. Notes', 30),
    ('bugs', 'Software Bugs 🐛', 40)
;


CREATE TABLE IF NOT EXISTS checkouts (
    time TIMESTAMP(3) UNIQUE NOT NULL,
    tagnumber INTEGER DEFAULT NULL,
    customer_name VARCHAR(48) DEFAULT NULL,
    customer_psid VARCHAR(24) DEFAULT NULL,
    checkout_bool BOOLEAN DEFAULT FALSE,
    checkout_date DATE DEFAULT NULL,
    return_date DATE DEFAULT NULL,
    checkout_group VARCHAR(48) DEFAULT NULL,
    note VARCHAR(512) DEFAULT NULL
);


DROP TABLE IF EXISTS static_emojis;
CREATE TABLE IF NOT EXISTS static_emojis (
    keyword VARCHAR(64) UNIQUE NOT NULL,
    regex VARCHAR(64) DEFAULT NULL,
    replacement VARCHAR(64) DEFAULT NULL,
    text_bool BOOLEAN DEFAULT NULL,
    case_sensitive_bool BOOLEAN DEFAULT NULL
);

INSERT INTO static_emojis (keyword, regex, replacement, text_bool, case_sensitive_bool) VALUES 
    (':)', '\:\)', '😀', NULL, NULL),
    (':D', '\:D\)', '😁', NULL, TRUE),
    (';)', '\;\)', '😉', NULL, NULL),
    (':P', '\:P', '😋', NULL, NULL),
    (':|', '\:\|', '😑', NULL, NULL),
    (':0', '\:0', '😲', NULL, NULL),
    (':O', '\:O', '😲', NULL, NULL),
    (':(', '\:\(', '😞', NULL, NULL),
    (':<', '\:\<', '😡', NULL, NULL),
    (':\', '\:\\', '😕', NULL, NULL),
    (';(', '\;\(', '😢', NULL, NULL),
    ('check', '\:check', '✅', TRUE, TRUE),
    ('done', '\:done', '✅', TRUE, TRUE),
    ('x', '\:x', '❌', TRUE, NULL),
    ('cancel', '\:cancel', '🚫', TRUE, TRUE),
    ('working', '\:working', '⌛', TRUE, TRUE),
    ('waiting', '\:waiting', '⌛', TRUE, TRUE),
    ('inprogress', '\:inprogress', '⌛', TRUE, TRUE),
    ('shurg', '\:shrug', '🤷', TRUE, TRUE),
    ('clock', '\:clock', '🕓', TRUE, TRUE),
    ('warning', '\:warning', '⚠️', TRUE, TRUE),
    ('arrow', '\:arrow', '⏩', TRUE, TRUE),
    ('bug', '\:bug', '🐛', TRUE, TRUE),
    ('poop', '\:poop', '💩', TRUE, TRUE),
    ('star', '\:star', '⭐', TRUE, TRUE),
    ('heart', '\:heart', '❤️', TRUE, TRUE),
    ('love', '\:love', '❤️', TRUE, TRUE),
    ('fire', '\:fire', '🔥', TRUE, TRUE),
    ('like', '\:like', '👍', TRUE, TRUE),
    ('dislike', '\:dislike', '👎', TRUE, TRUE),
    ('info', '\:info', 'ℹ️', TRUE, TRUE),
    ('pin', '\:pin', '📌', TRUE, TRUE),
    ('clap', '\:clap', '👏', TRUE, TRUE),
    ('celebrate', '\:celebrate', '🥳', TRUE, TRUE),
    ('hmm', '\:hmm', '🤔', TRUE, TRUE),
    ('alert', '\:alert', '🚨', TRUE, TRUE),
    ('mindblown', '\:mindblown', '🤯', TRUE, TRUE),
    ('shock', '\:shock', '⚡', TRUE, TRUE),
    ('wow', '\:wow', '😲', TRUE, TRUE),
    ('eyes', '\:eyes', '👀', TRUE, TRUE),
    ('looking', '\:looking', '👀', TRUE, TRUE)
;