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
	job_name_readable: string | null;
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

type AllJobs = {
	job_name: string;
	job_name_readable: string;
	job_sort_order: number;
	job_hidden: boolean;
};

const updateOnlineJobQueueForm = document.getElementById('update-all-online-jobs-form') as HTMLFormElement | null;
const updateOnlineJobQueueSelect = document.getElementById('update-all-online-jobs-select') as HTMLSelectElement | null;
const updateOnlineJobQueueButton = document.getElementById('update-all-online-jobs-submit') as HTMLButtonElement | null;
const onlineClientsDiv = document.getElementById('online-clients') as HTMLDivElement | null;
const offlineClientsDiv = document.getElementById('offline-clients') as HTMLDivElement | null;

let jobQueueInterval: number;

document.addEventListener('DOMContentLoaded', async () => {
	const allJobs = await getAllJobs();
	if (updateOnlineJobQueueSelect) {
		for (const job of allJobs) {
			if (job.job_hidden) {
				continue;
			}
			const option = document.createElement('option');
			option.value = job.job_name;
			option.textContent = job.job_name_readable;
			updateOnlineJobQueueSelect.appendChild(option);
		}
	}

	// Initial fetch and set interval for realtime updates
	getJobQueueData();
	jobQueueInterval = setInterval(() => {
		getJobQueueData();
	}, 10000);
});

async function getAllJobs(): Promise<AllJobs[]> {
	try {
		const data: AllJobs[] = await fetchData('/api/job_queue/all_jobs', false);
		if (Array.isArray(data)) {
			return data;
		} else {
			console.error('Expected array but got:', data);
			return [];
		}
	} catch (error) {
		console.error('Error fetching all jobs:', error);
		return [];
	}
}
		

if (updateOnlineJobQueueForm && updateOnlineJobQueueSelect && updateOnlineJobQueueButton) {
	updateOnlineJobQueueSelect.addEventListener('change', () => {
		updateOnlineJobQueueButton.disabled = updateOnlineJobQueueSelect.value === '';
	});

	updateOnlineJobQueueForm.addEventListener('submit', async (event) => {
		event.preventDefault();
		updateOnlineJobQueueSelect.disabled = true;
		updateOnlineJobQueueButton.disabled = true;

		const selectedValue = updateOnlineJobQueueSelect.value;
		if (!selectedValue) {
			alert('Please select a valid job to queue.');
			updateOnlineJobQueueSelect.disabled = false;
			updateOnlineJobQueueButton.disabled = false;
			return;
		}

		try {
			const allJobsArr: AllJobs[] = await getAllJobs();
			if (!allJobsArr || !Array.isArray(allJobsArr) || allJobsArr.length === 0) {
				throw new Error('Failed to get available jobs from ' + '/api/job_queue/all_jobs');
			}

			for (const job of allJobsArr) {
				if (job.job_name === selectedValue) {
					let clientJob: AllJobs = {
						job_name: job.job_name,
						job_name_readable: job.job_name_readable,
						job_sort_order: job.job_sort_order,
						job_hidden: job.job_hidden
					};
					const response = await fetch('/api/job_queue/update_all_online_clients', {
						method: 'POST',
						headers: {
							'Content-Type': 'application/json',
						},
						body: JSON.stringify(clientJob),
					});
					if (!response.ok) {
						throw new Error('Server responded with ' + response.status);
					}
				}
			}

		} catch (error) {
			console.error('Error updating all online clients with the selected job:', error);
			alert('An error occurred while updating all online clients with the selected job.');
		} finally {
			updateOnlineJobQueueSelect.disabled = false;
			updateOnlineJobQueueButton.disabled = false;
		}
	});
}

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
	if (!onlineClientsDiv || !offlineClientsDiv) return;

	const onlineTableFragment = document.createDocumentFragment();
	const offlineTableFragment = document.createDocumentFragment();

	for (const entry of data) {
		if (!entry.tagnumber) continue;

		const clientRow = document.createElement('div');
		clientRow.id = `client-row-${entry.tagnumber}`;
		// clientRow.dataset.tagnumber = entry.tagnumber.toString();
		clientRow.className = 'client-row-container';

		const gridContainer = document.createElement('div');
		gridContainer.className = 'client-row-grid';

		const col1 = document.createElement('div');
		col1.className = 'grid-item headers';

		const tagNum = document.createElement('p');
		tagNum.style.fontWeight = 'bold';
		tagNum.textContent = entry.tagnumber !== null ? entry.tagnumber.toString() : 'N/A';
		col1.appendChild(tagNum);

		const serialNumberLabel = document.createElement('p');
		serialNumberLabel.style.fontStyle = 'italic';
		serialNumberLabel.textContent = 'Serial Number: ';
		col1.appendChild(serialNumberLabel);
		const serialNumber = document.createElement('span');
		serialNumber.textContent = entry.system_serial || 'N/A';
		serialNumberLabel.appendChild(serialNumber);

		const manufacturerModelLabel = document.createElement('p');
		manufacturerModelLabel.style.fontStyle = 'italic';
		manufacturerModelLabel.textContent = 'Manufacturer/Model: ';
		col1.appendChild(manufacturerModelLabel);
		const manufacturerModel = document.createElement('span');
		manufacturerModel.textContent = `${entry.system_manufacturer || 'N/A'} - ${entry.system_model || 'N/A'}`;
		manufacturerModelLabel.appendChild(manufacturerModel);

		const locationLabel = document.createElement('p');
		locationLabel.style.fontStyle = 'italic';
		locationLabel.textContent = 'Location: ';
		col1.appendChild(locationLabel);
		const location = document.createElement('span');
		location.textContent = entry.location || 'N/A';
		locationLabel.appendChild(location);

		const departmentLabel = document.createElement('p');
		departmentLabel.style.fontStyle = 'italic';
		departmentLabel.textContent = 'Department: ';
		col1.appendChild(departmentLabel);
		const department = document.createElement('span');
		department.textContent = entry.department_name || 'N/A';
		departmentLabel.appendChild(department);

		const statusLabel = document.createElement('p');
		statusLabel.style.fontStyle = 'italic';
		statusLabel.textContent = 'Status: ';
		col1.appendChild(statusLabel);
		const status = document.createElement('span');
		status.textContent = entry.client_status || 'N/A';
		statusLabel.appendChild(status);

		gridContainer.appendChild(col1);

		if (entry.online) {
			onlineTableFragment.appendChild(clientRow);
		} else {
			offlineTableFragment.appendChild(clientRow);
		}
	}
	onlineClientsDiv.innerHTML = '';
	onlineClientsDiv.appendChild(onlineTableFragment);
	
	offlineClientsDiv.innerHTML = '';
	offlineClientsDiv.appendChild(offlineTableFragment);
}
