// Job Queue TypeScript File
type JobQueueEntry = {
	tagnumber: number | null
	system_serial: string | null
	os_installed: string | null
	os_name: string | null
	kernel_updated: boolean | null
	bios_updated: boolean | null
	bios_version: string | null
	system_manufacturer: string | null
	system_model: string | null
	battery_charge: number | null
	battery_status: string | null
	cpu_temp: number | null
	disk_temp: number | null
	max_disk_temp: number | null
	power_usage: number | null
	network_usage: number | null
	client_status: string | null
	is_broken: boolean | null
	job_queued: boolean | null
	queue_position: number | null
	job_active: boolean | null
	job_name: string | null
	job_status: string | null
	job_clone_mode: string | null
	job_erase_mode: string | null
	last_job_time: Date | null
	location: string | null
	last_heard: Date | null
	uptime: number | null
	online: boolean | null
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

function updateJobQueueTable(data: JobQueueEntry[]) {
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
