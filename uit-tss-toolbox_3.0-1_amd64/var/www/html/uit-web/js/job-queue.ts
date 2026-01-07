// Job Queue TypeScript File
type JobQueueEntry = {
	tagnumber: number
	system_serial: string
	os_installed: string
	os_name: string
	kernel_updated: boolean
	bios_updated: boolean
	bios_version: string
	system_manufacturer: string
	system_model: string
	battery_charge: number
	battery_status: string
	cpu_temp: number
	disk_temp: number
	max_disk_temp: number
	power_usage: number
	network_usage: number
	client_status: string
	is_broken: boolean
	job_queued: boolean
	queue_position: number
	job_active: boolean
	job_name: string
	job_status: string
	job_clone_mode: string
	job_erase_mode: string
	last_job_time: Date
	location: string
	last_heard: Date
	uptime: number
	online: boolean
};

getJobQueueData();

setInterval(() => {
		getJobQueueData();
}, 10000);

async function getJobQueueData() {
		try {
				const response = fetchData('/api/job_queue/overview', true);
				const data: JobQueueEntry[] = await response;
				updateJobQueueTable(data);
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

		data.forEach((entry) => {
				const row = document.createElement('tr');
				const tagCell = document.createElement('td');
				tagCell.textContent = entry.tagnumber.toString();
				row.appendChild(tagCell);

				if (entry.online) {
						onlineTableFragment.appendChild(row);
				} else {
						offlineTableFragment.appendChild(row);
				}
		});
		onlineTableBody.innerHTML = '';
		onlineTableBody.appendChild(onlineTableFragment);
		
		offlineTableBody.innerHTML = '';
		offlineTableBody.appendChild(offlineTableFragment);
}
