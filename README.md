# API endpoints (WIP)
> [!IMPORTANT]
> Not all endpoints are implemented and/or actively used. Use API with caution.

> [!IMPORTANT]
> Unless otherwise noted, a GET request will always return all other variables, regardless of request type. State-chaning requests (POST, DELETE, PUSH, etc.) requests will usually only return an error, unless otherwise noted with "Output POST Variables".

## To-do
- [ ] Implement all WIP endpoints
  - [ ] GET endpoints
  - [ ] POST endpoints
  - [ ] PUT endpoints
  - [ ] DELETE endpoints
- [ ] Double-check all return types
- [ ] Test and implement all endpoints

# General Data
## Server time
### **/api/server_time[?...]**
<details open>
<summary>Input GET Variables</summary>

| Variable | Type | Null |
|---------:|------|------|
|null      |null  |Yes   |

</details>

<details open>
<summary>Output GET Variables</summary>

| Variable   | Type                                              | Null |
|-----------:|---------------------------------------------------|------|
|client_time |*time string formatted (YYYY-MM-DD hh:mm:ss.vvv)*  |No    |

</details>

<details open>
<summary>Input POST Variables</summary>

| Variable  | Type                                              | Null |
|-----------|---------------------------------------------------|------|
|client_time|*time string formatted (YYYY-MM-DD hh:mm:ss.vvv)*  |No    |

</details>

<details open>
<summary>Output POST Variables</summary>

| Variable                | Type      |
|------------------------:|-----------|
| client_has_correct_time | *boolean* |

</details>

> [!NOTE]
> *client_time* (POST only) returns a boolean if *client_time* is within margin of error of the server's time
## Reverse Client Lookup
- **<ins>/api/lookup[?...]</ins>**
- GET variables:

|   Variable   |   Type   |  Null  |
|--------------|----------|--------|
|tagnumber     | *int*    | Yes/No |
|system_serial | *string* | Yes/No |

> [!NOTE] 
> If GETing data, you must specify the *tagnumber* OR *system_serial*, not both


## Hardware Data
### Overall Disk Health By Date
- **<ins>/api/hardware/overview/disk_health[?...]</ins>**
- GET variables:
  - *date*       [GET:  date string, RET: float]
- POST variables:
  - *date*       [POST: date string, RET: null]
  - *percentage* [POST: float,       RET: err]
> [!NOTE]
> If updating the percentage for a given date, **both** the *date* and *percentage* have to be specified in the POST body request.

### Overall Battery Health By Date
- **<ins>/api/hardware/battery_health[?...]</ins>**
- GET variables:
  - *date*       [GET:  date string, RET: float]
- POST variables:
  - *date*       [POST: date (string), RET: null]
  - *percentage* [POST: float,         RET: err]
> [!NOTE]
> If updating the percentage for a given date, **both** the *date* and *percentage* have to be specified in the POST body request.

general hardware data by tag
- GET/POST <ins>/api/client/hardware[?...]</ins>
- GET/POST variables: 
>tagnumber                  [GET:  integer, RET: all others]*
[POST: integer, RET: err]**
>ethernet_mac               [POST: string,  RET: err]
>wifi_mac                   [POST: string,  RET: err]
>model                      [POST: string,  RET: err]
>uuid (system UUID in DMI)  [POST: string,  RET: err]
>SKU                        [POST: string,  RET: err]
>chassis_type               [POST: string,  RET: err]
>motherboard_manufacturer   [POST: string,  RET: err]
>motherboard_serial         [POST: string,  RET: err]
>system_manufacturer        [POST: string,  RET: err]
-note: like many other endpoints, if you GET a certain variable,
all other variables will be returned (hence, "all others").

CPU data by tag
- GET/POST (/api/client/hardware/cpu[?...])
- GET/POST variables:
>tagnumber                  [GET:  integer, RET: all others]*
[POST: integer, RET: err]**
>cpu_manufacturer           [POST: string,  RET: err]
>cpu_model                  [POST: string,  RET: err]
>cpu_max_speed              [POST: float,   RET: err]
>cpu_cores                  [POST: integer, RET: err]
>cpu_threads                [POST: integer, RET: err]
-note: like many other endpoints, if you GET a certain variable,
all other variables will be returned (hence, "all others").

disk data by tag
- GET/POST (/api/client/hardware/disk[?...])
- GET/POST variables: 
>tagnumber*                 [GET:  integer, RET: all others]*
[POST: integer, RET: err]**
>name (in linux e.g. sda)   [POST: string,  RET: err]
>model                      [POST: string,  RET: err]
>serial                     [POST: string,  RET: err]
>firmware                   [POST: string,  RET: err]
>temp                       [POST: integer, RET: err]
>size (capacity in GB)      [POST: integer, RET: err]
>reads (total)              [POST: integer, RET: err]
>writes (total)             [POST: integer, RET: err]
>power_on_hours             [POST: integer, RET: err]
>power_cycles               [POST: integer, RET: err]
>type                       [POST: string,  RET: err]
>generic_error              [POST: string,  RET: err]
>data_integrity_error       [POST: string,  RET: err]
>total_error_count          [POST: integer, RET: err]
-note: like many other endpoints, if you GET a certain variable,
all other variables will be returned (hence, "all others").

battery data by tag
- GET/POST (/api/client/hardware/battery[?...])
- GET/POST variables: 
>tagnumber          [GET:  integer,  RET: all others]*
[POST: integer,  RET: err]**
>current_charge     [POST: integer,  RET: err]
>factory_charge     [POST: integer,  RET: err]
>overall_health     [POST: float,    RET: err]
>charge_cycles      [POST: integer,  RET: err]
>model              [POST: string,   RET: err]
>capactiy           [POST: integer,  RET: err]
>serial             [POST: string,   RET: err]
>manufacturer       [POST: string,   RET: err]
-note: like many other endpoints, if you GET a certain variable,
all other variables will be returned (hence, "all others").

RAM/memory data by tag
- GET/POST (/api/client/hardware/memory[?...])
- GET/POST variables: 
>tagnumber  [GET:  integer, RET: all others]*
[POST: integer, RET: err]**
>serial     [POST: string,  RET: err]
>capacity   [POST: integer, RET: err]
>speed      [POST: integer, RET: err]
-note: RAM serial for 2+ sticks is concatenated 
into one string where each stick's serial
number is separated by two dashes (--)
-note: like many other endpoints, if you GET a certain variable,
all other variables will be returned (hence, "all others").

Hardware temps by tag
- GET/POST (/api/client/hardware/temp/cpu[?...])
- GET/POST variables:
>tagnumber [GET:  integer,  RET: float]
[GET:  null,     RET: cpu_min/cpu_max/cpu_avg]
[POST: integer,  RET: err]**
>cpu       [POST: float,    RET: err] 
>cpu_min   [Readonly,       RET: float]
>cpu_max   [Readonly,       RET: float]
>cpu_avg   [Readonly,       RET: float]

- GET/POST (/api/client/hardware/temp/disk[?...])
- GET/POST variables:
>tagnumber [GET:  integer,  RET: float]
[GET:  null,     RET: disk_min/disk_max/disk_avg]
[POST: integer,  RET: err]**
>disk      [POST: float,    RET: err]
>disk_min  [Readonly,       RET: float]
>disk_max  [Readonly,       RET: float]
>disk_avg  [Readonly,       RET: float]
-note: if no GET tagnumber query is specified, 
then the min/max/avg temps are returned
for all online clients
-note: like many other endpoints, if you GET a certain variable,
all other variables will be returned (hence, "all others").


-- Software Data --
BIOS data
- GET/POST (/api/client/software/bios[?...])
- GET/POST variables: 
>tagnumber          [GET:  integer,  RET: all others]*
[POST: integer,  RET: err]**
>bios_version       [POST: string,   RET: err]
>bios_date          [POST: date,     RET: err]
>bios_revision      [POST: string,   RET: err]
-note: like many other endpoints, if you GET a certain variable,
all other variables will be returned (hence, "all others").

OS data
- GET/POST (/api/client/software/os[?...])
- GET/POST variables: 
>tagnumber          [GET:  integer,   RET: all others]*
[POST: integer,   RET: err]**
>os_installed       [POST: boolean,   RET: err]
>os_name            [POST: string,    RET: err]
>os_version         [POST: string,    RET: err]
>os_installed_time  [POST: datetime,  RET: err]
>tpm_version        [POST: string,    RET: err]
>boot_time          [POST: integer,   RET: err]
-note: like many other endpoints, if you GET a certain variable,
all other variables will be returned (hence, "all others").



-- Job Queue Data --
job queue and queue position data for clients
- GET (/api/job_queue/overview[?...])
- GET variables: total_queued_jobs, 
total_active_jobs,
total_active_blocking_jobs
-blocking jobs are jobs that have to be
queued, e.g. clone jobs.
-job active by tag
- GET/POST (/api/job_queue/client/queued_job[?...])
- GET/POST variables: tagnumber*, queued_job, 
job_active, queue_position

jobs available by tag
- GET (/api/job_queue/client/job_available[?...])
- GET variables: tagnumber*, job_name
-note: job_name returns boolean if job_name is 
or is not available for tag number. Tag
must be specified if job_name is specified.

all possible jobs
- GET (/api/job_queue/overview/all_jobs)
- GET variables:
>tagnumber  [GET: integer, RET: all others]
>job_name   [GET: null,    RET: comma separated string]              


-- Job control
uuid, time, date, location, job_type, erase_mode, clone_mode, erase_time, clone_time, clone_master, image_name, image_name_formatted, 


-- Job Queue Data (server/client side) --
remote job queue data
-online clients
- GET (/api/job_queue/overview/online)
- GET variables:
>tagnumber      [GET: integer, RET: all others]
>present        [Readonly,     RET: datetime]
>present_bool   [Readonly,     RET: boolean]
>uptime         [Readonly,     RET: duration]
>status         [Readonly,     RET: string]
>job_queued     [Readonly,     RET: string]
>job_active     [Readonly,     RET; boolean] 
>queue_position [Readonly,     RET: integer]
-update queued job for all online clients
- POST (/api/job_queue/overview/online)
-offline clients
- GET (/api/job_queue/overview/offline)

job queue data by tag
-online/offline, current status & queued job
- GET (/api/job_queue/client?tagnumber=&formatted=)
- GET (/api/job_queue/client?serial=&formatted=)
-update queued job
- POST (/api/job_queue/client?tagnumber=)
- POST (/api/job_queue/client?serial=)

-- Checkout Data --
All checkouts
-currently checked out
- GET (/api/checkouts/current)
-historical checkouts
- GET (/api/checkouts/all-time)
-for each client
-currently checked out
- GET (/api/checkouts/current?tagnumber=)
-historical log for a given client
- GET (/api/checkouts/all-time?tagnumber=)

-- Location Data --
list of locations and number of entries data
- GET (/api/locations/overview)

location by tagnumber
- GET (/api/locations/client[?...])
- GET variables:
>tagnumber      [GET: integer,  RET: all others]
[POST: integer, RET: err]
>system_serial  [POST: string,  RET: err]
>location       [POST: string,  RET: err]
>department     [POST: string,  RET: err]
>ad_domain      [POST: string,  RET: err]
>working        [POST: boolean, RET: err]
>note           [POST: string,  RET: err]
>disk_removed   [POST: boolean, RET: err]

-- Notes data --