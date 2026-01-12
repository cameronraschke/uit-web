// Job Queue TypeScript File
type JobQueueTableRow = {
	tagnumber: number | null;
	system_serial: string | null;
	system_manufacturer: string | null;
	system_model: string | null;
	location: string | null;
	department_name: string | null;
	client_status: string | null;
	is_broken: boolean | null;
	disk_removed: boolean | null;
	temp_warning: boolean | null;
	battery_health_warning: boolean | null;
	checkout_bool: boolean | null;
	kernel_updated: boolean | null;
	last_heard: Date | null;
	uptime: number | null;
	online: boolean | null;
	job_active: boolean | null;
	job_queued: boolean | null;
	queue_position: number | null;
	job_name: string | null;
	job_clone_mode: string | null;
	job_erase_mode: string | null;
	job_status: string | null;
	last_job_time: Date | null;
	os_installed: string | null;
	os_name: string | null;
	os_updated: boolean | null;
	domain_joined: boolean | null;
	domain_name: string | null;
	bios_updated: boolean | null;
	bios_version: string | null;
	cpu_usage: number | null;
	cpu_temp: number | null;
	cpu_temp_warning: boolean | null;
	ram_usage: number | null;
	ram_capacity: number | null;
	disk_usage: number | null;
	disk_temp: number | null;
	disk_type: string | null;
	disk_size: number | null;
	max_disk_temp: number | null;
	disk_temp_warning: boolean | null;
	network_link_status: string | null;
	network_link_speed: number | null;
	network_usage: number | null;
	battery_charge: number | null;
	battery_status: string | null;
	battery_health: number | null;
	plugged_in: boolean | null;
	power_usage: number | null;
};

let jobQueueInterval: number;
getJobQueueData();
jobQueueInterval = setInterval(() => {
	getJobQueueData();
}, 10000);

document.addEventListener('visibilitychange', () => {
	clearInterval(jobQueueInterval);
	if (document.visibilityState === 'visible') {
		getJobQueueData();
		jobQueueInterval = setInterval(() => {
			getJobQueueData();
		}, 10000);
	}
});


async function getJobQueueData() {
	try {
		const data = await fetchData('/api/job_queue/overview', false);
		if (Array.isArray(data)) {
			updateJobQueueTable(data);
		} else {
			console.error('Expected array but got:', data);
		}
	} catch (error) {
		console.error('Error fetching job queue data:', error);
	}
}

function updateJobQueueTable(data: JobQueueTableRow[]) {
	const onlineTableBody = document.querySelector('#online-clients-table tbody');
	const offlineTableBody = document.querySelector('#offline-clients-table tbody');
	if (!onlineTableBody || !offlineTableBody) return;

	const onlineTableFragment = document.createDocumentFragment();
	const offlineTableFragment = document.createDocumentFragment();

	for (const entry of data) {
		const row = document.createElement('tr');
		const tagCell = document.createElement('td');
		const tag = entry.tagnumber !== null ? entry.tagnumber.toString() : 'N/A';
		const tagLink = document.createElement('a');
		const tagURL = new URL(window.location.origin + '/client');
		tagURL.searchParams.append('tagnumber', tag);
		tagLink.textContent = tag;
		tagLink.href = tagURL.toString();
		tagLink.target = '_blank';
		tagCell.appendChild(tagLink);
		row.appendChild(tagCell);

		if (entry.online) {
			onlineTableFragment.appendChild(row);
		} else {
			offlineTableFragment.appendChild(row);
		}
	}
	onlineTableBody.innerHTML = '';
	onlineTableBody.appendChild(onlineTableFragment);
	
	offlineTableBody.innerHTML = '';
	offlineTableBody.appendChild(offlineTableFragment);
}
