-- SQL user creation
CREATE OR REPLACE PROCEDURE sqlCreateUsers()
LANGUAGE SQL
AS $$

CREATE USER cameron WITH SUPERUSER CREATEDB CREATEROLE PASSWORD 'UHouston!';

CREATE USER uitclient PASSWORD 'UHouston!';

CREATE USER uitweb PASSWORD 'UHouston!';

$$;

-- SQL permissions
CREATE OR REPLACE PROCEDURE sqlPermissions()
LANGUAGE SQL
AS $$

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO cameron WITH GRANT OPTION;
GRANT EXECUTE ON ALL PROCEDURES IN SCHEMA public TO cameron;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO cameron;

GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO uitclient;
GRANT EXECUTE ON ALL PROCEDURES IN SCHEMA public TO uitclient;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO uitclient;

GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO uitweb;
GRANT DELETE ON client_images TO uitweb;
GRANT EXECUTE ON ALL PROCEDURES IN SCHEMA public TO uitweb;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO uitweb;

$$;



-- Select remote table data
CREATE OR REPLACE FUNCTION selectRemote()
RETURNS TABLE (
  "Tag" INTEGER,
  "Last Heard" TEXT,
  "Location" VARCHAR,
  "Last Job Time" TEXT,
  "Pending Job" VARCHAR,
  "Status" VARCHAR,
  "Kernel/BIOS Updated" TEXT,
  "OS Name" VARCHAR,
  "Battery Status" TEXT,
  "Uptime" TEXT,
  "CPU Temp/Disk Temp/Watts" TEXT
) AS $$
BEGIN
    RETURN QUERY 
    SELECT remote.tagnumber, 
        TO_CHAR(remote.present, 'MM/DD/YY HH12:MI:SS AM'), locationFormatting(t3.location), 
        TO_CHAR(remote.last_job_time, 'MM/DD/YY HH12:MI:SS AM'), 
        remote.job_queued AS "Pending Job", remote.status AS "Status",
        CONCAT((CASE WHEN remote.kernel_updated = TRUE THEN 'Yes' ELSE 'No' END), '/', (CASE WHEN client_health.bios_updated = TRUE THEN 'Yes' ELSE 'No' END)), client_health.os_name, 
        CONCAT(remote.battery_charge, '% (', remote.battery_status, ')'), 
        TO_CHAR(NOW() - TO_TIMESTAMP(EXTRACT(EPOCH FROM NOW()) - (EXTRACT(EPOCH FROM NOW()) - EXTRACT(EPOCH FROM (NOW() - (remote.uptime || ' second')::interval)))), 'DDD"d," HH24"h" MI"m" SS"s"'), 
        CONCAT(remote.cpu_temp, '°C', '/' , remote.disk_temp, '°C', '/', remote.watts_now, ' Watts')
      FROM remote 
      LEFT JOIN (SELECT s1.time, s1.tagnumber FROM (SELECT time, tagnumber, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS "row_nums" FROM locations) s1 WHERE s1.row_nums = 1) t1
        ON remote.tagnumber = t1.tagnumber
      LEFT JOIN client_health ON remote.tagnumber = client_health.tagnumber
      LEFT JOIN (SELECT tagnumber, location, row_nums FROM (SELECT tagnumber, location, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS "row_nums" FROM locations) s3 WHERE s3.row_nums = 1) t3
        ON t3.tagnumber = remote.tagnumber
      LEFT JOIN (SELECT tagnumber, queue_position FROM (SELECT tagnumber, ROW_NUMBER() OVER (ORDER BY tagnumber ASC) AS "queue_position" FROM remote WHERE job_queued IS NOT NULL) s2) t2
        ON remote.tagnumber = t2.tagnumber
      WHERE remote.present_bool = TRUE
      ORDER BY
        (CASE WHEN remote.status LIKE 'fail%' THEN 1 ELSE 0 END) DESC, job_queued IS NULL ASC, job_active DESC, queue_position ASC,
        (CASE WHEN job_queued = 'data collection' THEN 20 WHEN job_queued = 'update' THEN 15 WHEN job_queued = 'nvmeVerify' THEN 14 WHEN job_queued =  'nvmeErase' THEN 12 WHEN job_queued =  'hpCloneOnly' THEN 11 WHEN job_queued = 'hpEraseAndClone' THEN 10 WHEN job_queued = 'findmy' THEN 8 WHEN job_queued = 'shutdown' THEN 7 WHEN job_queued = 'fail-test' THEN 5 ELSE NULL END) DESC, 
        (CASE WHEN status = 'Waiting for job' THEN 1 ELSE 0 END) ASC, (CASE WHEN client_health.os_installed = TRUE THEN 1 ELSE 0 END) DESC, (CASE WHEN remote.kernel_updated = TRUE THEN 1 ELSE 0 END) DESC, (CASE WHEN client_health.bios_updated = TRUE THEN 1 ELSE 0 END) DESC, remote.last_job_time DESC;
    END
    $$ LANGUAGE plpgsql;


-- Select missing remote table data
CREATE OR REPLACE FUNCTION selectRemoteMissing()
RETURNS TABLE (
  "Tag" INTEGER,
  "Last Heard" TEXT,
  "Location" VARCHAR,
  "Last Job Time" TEXT,
  "Pending Job" VARCHAR,
  "Status" VARCHAR,
  "Kernel/BIOS Updated" TEXT,
  "OS Name" VARCHAR,
  "Battery Status" TEXT,
  "Uptime" TEXT,
  "CPU Temp/Disk Temp/Watts" TEXT
) AS $$
BEGIN
  RETURN QUERY 
    SELECT remote.tagnumber, 
        TO_CHAR(remote.present, 'MM/DD/YY HH12:MI:SS AM'), locationFormatting(t3.location), 
        TO_CHAR(remote.last_job_time, 'MM/DD/YY HH12:MI:SS AM'), 
        remote.job_queued AS "Pending Job", remote.status AS "Status",
        CONCAT((CASE WHEN remote.kernel_updated = TRUE THEN 'Yes' ELSE 'No' END), '/', (CASE WHEN client_health.bios_updated = TRUE THEN 'Yes' ELSE 'No' END)), client_health.os_name, 
        CONCAT(remote.battery_charge, '% (', remote.battery_status, ')'), 
        TO_CHAR(NOW() - TO_TIMESTAMP(EXTRACT(EPOCH FROM NOW()) - (EXTRACT(EPOCH FROM NOW()) - EXTRACT(EPOCH FROM (NOW() - (remote.uptime || ' second')::interval)))), 'DDD"d," HH24"h" MI"m" SS"s"'), 
        CONCAT(remote.cpu_temp, '°C', '/' , remote.disk_temp, '°C', '/', remote.watts_now, ' Watts')
      FROM remote 
      LEFT JOIN (SELECT s1.time, s1.tagnumber FROM (SELECT time, tagnumber, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS "row_nums" FROM locations) s1 WHERE s1.row_nums = 1) t1
        ON remote.tagnumber = t1.tagnumber
      LEFT JOIN client_health ON remote.tagnumber = client_health.tagnumber
      LEFT JOIN (SELECT tagnumber, location, row_nums FROM (SELECT tagnumber, location, ROW_NUMBER() OVER (PARTITION BY tagnumber ORDER BY time DESC) AS "row_nums" FROM locations) s3 WHERE s3.row_nums = 1) t3
        ON t3.tagnumber = remote.tagnumber
      LEFT JOIN (SELECT tagnumber, queue_position FROM (SELECT tagnumber, ROW_NUMBER() OVER (ORDER BY tagnumber ASC) AS "queue_position" FROM remote WHERE job_queued IS NOT NULL) s2) t2
        ON remote.tagnumber = t2.tagnumber
      WHERE remote.present_bool = FALSE AND remote.present IS NOT NULL
      ORDER BY remote.present DESC;
    END
  $$ LANGUAGE plpgsql;


-- Select remote table stats
CREATE OR REPLACE FUNCTION selectRemoteStats()
RETURNS TABLE (
  "Present Clients" BIGINT,
  "Avg. Battery Charge" TEXT,
  "Avg. CPU Temp" TEXT,
  "Avg. Disk Temp" TEXT,
  "Avg. Power Draw" TEXT,
  "OS's Installed" BIGINT
) AS $$
BEGIN
  RETURN QUERY 
    SELECT 
    (SELECT COUNT(remote.tagnumber) FROM remote WHERE remote.present_bool = TRUE),
    CONCAT(ROUND(AVG(remote.battery_charge), 0), '%'),
    CONCAT(ROUND(AVG(remote.cpu_temp), 1), '°C'),
    CONCAT(ROUND(AVG(remote.disk_temp), 1), '°C'),
    CONCAT(ROUND(AVG(remote.watts_now), 1), ' Watts'),
    COUNT(client_health.os_installed)
    FROM remote 
    LEFT JOIN client_health ON remote.tagnumber = client_health.tagnumber 
    WHERE remote.present_bool = TRUE;
    END
    $$ LANGUAGE plpgsql;



CREATE OR REPLACE PROCEDURE selectLocationAutocomplete()
LANGUAGE SQL
BEGIN ATOMIC
  SELECT MAX(t1.time) AS "time", t1.location, MAX(t1.row_nums) AS "row_nums" FROM (SELECT time, locationFormatting(REPLACE(REPLACE(REPLACE(location, '\\', '\\\\'), '''', '\\'''), '\"','\\"')) AS "location", ROW_NUMBER() OVER (PARTITION BY location ORDER BY time DESC) AS "row_nums" FROM locations WHERE time IN (SELECT MAX(time) FROM locations GROUP BY tagnumber)) t1 GROUP BY t1.location ORDER BY row_nums DESC;
END;


CREATE OR REPLACE FUNCTION locationFormatting(location VARCHAR(128)) 
RETURNS VARCHAR(128) AS $$
BEGIN
  RETURN CASE
    WHEN REGEXP_MATCH(location, '^.{1}$') IS NOT NULL THEN UPPER(location)
    WHEN REGEXP_MATCH(location, '^(checkout|check-out|check out)$') IS NOT NULL THEN 'Check Out'
    WHEN REGEXP_MATCH(location, '^(cam desk|cams desk|cam''s desk)$') IS NOT NULL THEN 'Cam''s Desk'
    WHEN REGEXP_MATCH(location, '^(matthew desk|matthews desk|matthew''s desk)$') IS NOT NULL THEN 'Matthew''s Desk'
    ELSE location
  END;
END;
$$ LANGUAGE plpgsql;