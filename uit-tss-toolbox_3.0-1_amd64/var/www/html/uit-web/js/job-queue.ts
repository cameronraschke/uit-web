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
	system_uptime: number | null;
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
	ad_domain: string | null;
	ad_domain_formatted: boolean | null;
	bios_updated: boolean | null;
	bios_version: string | null;
	cpu_usage: number | null;
	cpu_temp: number | null;
	cpu_temp_warning: boolean | null;
	memory_usage: number | null;
	memory_capacity: number | null;
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

type AllJobsCache = {
	jobs: AllJobs[] | null;
	timestamp: number;
};

const updateOnlineJobQueueForm = document.getElementById('update-all-online-jobs-form') as HTMLFormElement | null;
const updateOnlineJobQueueSelect = document.getElementById('update-all-online-jobs-select') as HTMLSelectElement | null;
const updateOnlineJobQueueButton = document.getElementById('update-all-online-jobs-submit') as HTMLButtonElement | null;
const onlineClientsDiv = document.getElementById('online-clients-container') as HTMLDivElement | null;
const offlineClientsDiv = document.getElementById('offline-clients-container') as HTMLDivElement | null;

let jobQueueInterval: number;

async function fetchAllJobs(purgeCache = false): Promise<AllJobs[]> {
	const cacheKey = 'uit_all_jobs';
	const cacheDuration = 5 * 60 * 1000; // 5 minutes in milliseconds
	const now = Date.now();
	if (!purgeCache) {
		const cachedDataString = localStorage.getItem(cacheKey);
		if (cachedDataString) {
			try {
				const cachedData: AllJobsCache = JSON.parse(cachedDataString);
				if (cachedData && cachedData.jobs && (now - cachedData.timestamp) < cacheDuration) {
					return cachedData.jobs;
				}
			} catch (error) {
				console.error('Error parsing cached data:', error);
			}
		}
	}

	try {
		const data: AllJobs[] = await fetchData('/api/job_queue/all_jobs', false);
		if (Array.isArray(data)) {
			localStorage.setItem(cacheKey, JSON.stringify({ jobs: data, timestamp: now }));
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
			const allJobsArr: AllJobs[] = await fetchAllJobs();
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
		fetchJobQueueData();
		jobQueueInterval = setInterval(() => {
			fetchJobQueueData();
		}, 10000);
	}
});


async function fetchJobQueueData() : Promise<JobQueueTableRow[] | []> {
	try {
		const data: JobQueueTableRow[] = await fetchData('/api/job_queue/overview', false);
		if (Array.isArray(data) && data.length > 0) {
			return data;
		} else {
			console.error('Expected array but got:', data);
			return [];
		}
	} catch (error) {
		console.error('Error fetching job queue data:', error);
		return [];
	}
}

function renderJobQueueTable(data: JobQueueTableRow[]) {
	if (!data || !Array.isArray(data) || data.length === 0) {
		console.warn('No job queue data to render.');
		return;
	}
	if (!onlineClientsDiv || !offlineClientsDiv) return;

	const onlineTableFragment = document.createDocumentFragment();
	const offlineTableFragment = document.createDocumentFragment();

	for (const entry of data) {
		if (entry && entry.online !== null && !entry.online) continue;
		if (!entry.tagnumber) {
			console.warn("No tagnumber for entry");
			continue;
		}

		const clientEntryContainer = document.createElement('div');
		clientEntryContainer.classList.add('client-row-container');
		clientEntryContainer.dataset.tagnumber = entry.tagnumber.toString();

		const clientGridContainer = document.createElement('div');
		clientGridContainer.className = 'client-row-grid';

		// Client identifiers and info
		const clientIdentifiers = document.createElement('div');
		clientIdentifiers.className = 'grid-item headers';
		const tagLabel = document.createElement('p');
		tagLabel.textContent = 'Tag Number: ';
		const tagAnchor = document.createElement('a');
		const tagURL = new URL(`/client`, window.location.origin);
		tagURL.searchParams.append('tagnumber', entry.tagnumber.toString());
		tagAnchor.href = tagURL.toString();
		tagAnchor.target = '_blank';
		tagAnchor.textContent = entry.tagnumber !== null ? `${entry.tagnumber.toString()}` : 'N/A';
		tagLabel.appendChild(tagAnchor);
		clientIdentifiers.appendChild(tagLabel);
		const serialNumber = document.createElement('p');
		serialNumber.textContent = `Serial Number: ${entry.system_serial || 'N/A'}`;
		clientIdentifiers.appendChild(serialNumber);
		const manufacturerModel = document.createElement('p');
		manufacturerModel.textContent = `Manufacturer/Model: ${entry.system_manufacturer || 'N/A'} - ${entry.system_model || 'N/A'}`;
		clientIdentifiers.appendChild(manufacturerModel);
		const location = document.createElement('p');
		location.textContent = `Location: ${entry.location || 'N/A'}`;
		clientIdentifiers.appendChild(location);
		const department = document.createElement('p');
		department.textContent = `Department: ${entry.department_name || 'N/A'}`;
		clientIdentifiers.appendChild(department);
		const status = document.createElement('p');
		status.textContent = `Status: ${entry.client_status || 'N/A'}`;
		clientIdentifiers.appendChild(status);

		// Live view
		const liveViewContainer = document.createElement('div');
		liveViewContainer.classList.add('grid-item', 'live-view-container');
		const liveViewHeader = document.createElement('p');
		liveViewHeader.style.fontStyle = 'italic';
		liveViewHeader.textContent = 'Live View: ';
		liveViewContainer.appendChild(liveViewHeader);
		const liveViewScreenshotContainer = document.createElement('div');
		liveViewScreenshotContainer.classList.add('image-container');
		const liveViewOffline = document.createElement('h2');
		liveViewOffline.textContent = 'Offline';
		liveViewScreenshotContainer.appendChild(liveViewOffline);
		liveViewContainer.appendChild(liveViewScreenshotContainer);
		const lastHeard = document.createElement('p');
		lastHeard.textContent = entry.last_heard ? `Last Heard: ${new Date(entry.last_heard).toLocaleString()}` : 'Last Heard: N/A';
		liveViewContainer.appendChild(lastHeard);
		clientEntryContainer.appendChild(clientGridContainer);

		// Job info
		const jobInfoContainer = document.createElement('div');
		jobInfoContainer.classList.add('grid-item');
		const jobName = document.createElement('p');
		jobName.textContent = `Job Queued: ${entry.job_name_readable || 'N/A'}`;
		jobInfoContainer.appendChild(jobName);
		const jobStatus = document.createElement('p');
		jobStatus.textContent = `Job Status: ${entry.job_status || 'N/A'}`;
		jobInfoContainer.appendChild(jobStatus);
		const clientUptime = document.createElement('p');
		if (entry.system_uptime !== null) {
			const uptimeSec = entry.system_uptime;
			const uptimeMins = Math.floor(uptimeSec / 60);
			const uptimeHours = Math.floor(uptimeMins / 60);
			const uptimeDays = Math.floor(uptimeHours / 24);
			if (uptimeDays > 0) {
				clientUptime.textContent = `Uptime: ${uptimeDays}d ${uptimeHours % 24}h ${uptimeMins % 60}m`;
			} else if (uptimeHours > 0) {
				clientUptime.textContent = `Uptime: ${uptimeHours}h ${uptimeMins % 60}m`;
			} else {
				clientUptime.textContent = `Uptime: ${uptimeMins}m ${uptimeSec % 60}s`;
			}
		} else {
			clientUptime.textContent = 'Uptime: N/A';
		}
		jobInfoContainer.appendChild(clientUptime);
		const jobSelectContainer = document.createElement('div');
		jobSelectContainer.classList.add('flex-container', 'horizontal');
		const jobSelect = document.createElement('select');
		fetchAllJobs().then(jobs => {
			const defaultOption = document.createElement('option');
			defaultOption.value = '';
			defaultOption.textContent = 'Select job to queue';
			defaultOption.disabled = true;
			defaultOption.selected = true;
			jobSelect.appendChild(defaultOption);
			for (const job of jobs) {
				if (job.job_hidden) {
					continue;
				}
				const option = document.createElement('option');
				option.value = job.job_name;
				option.textContent = job.job_name_readable;
				jobSelect.appendChild(option);
			}
		}).catch(error => {
			console.error('Error fetching all jobs for select dropdown:', error);
		});
		jobInfoContainer.appendChild(jobSelect);
		const queueJobButton = document.createElement('button');
		if (entry.job_queued || entry.job_name === "cancel") {
			queueJobButton.textContent = 'Cancel Job';
			queueJobButton.classList.add('svg-button', 'cancel');
		} else {
			queueJobButton.textContent = 'Queue Job';
			queueJobButton.classList.remove('svg-button', 'cancel');
		}
		queueJobButton.removeEventListener('click', async () => {});
		queueJobButton.addEventListener('click', async () => {
			let selectedJob = jobSelect.value || null;
			if (queueJobButton.classList.contains('svg-button.cancel')) {
				selectedJob = "cancel";
			} else {
				alert('Please select a job to queue.');
				return;
			}
			
			try {
				if (entry.tagnumber === null || selectedJob === null) {
					throw new Error('tagnumber or selected job is null');
				}
				await updateClientJob(entry.tagnumber, selectedJob);
			} catch (error) {
				console.error('Error queueing job:', error);
				alert('An error occurred while queueing the job. Please try again.');
			} finally {
				await initializeJobQueuePage();
			}
		});
		jobSelectContainer.appendChild(jobSelect);
		jobSelectContainer.appendChild(queueJobButton);
		if (entry.online) jobInfoContainer.appendChild(jobSelectContainer);
		if (entry.job_queued && entry.queue_position !== null) {
			jobSelect.value = entry.job_name || '';
			const queuePosition = document.createElement('p');
			queuePosition.textContent = `Queue Position: ${entry.queue_position}, last job completed at ${entry.last_job_time ? new Date(entry.last_job_time).toLocaleString() : 'N/A'}`;
			jobInfoContainer.appendChild(queuePosition);
		}

		// Software info
		const softwareInfoContainer = document.createElement('div');
		softwareInfoContainer.classList.add('grid-item');
		const osInfo = document.createElement('p');
		osInfo.textContent = `OS: ${entry.os_name || 'N/A'} ${entry.os_updated ? '(Updated)' : '(Not Updated)'}`;
		softwareInfoContainer.appendChild(osInfo);
		const domainJoined = document.createElement('p');
		domainJoined.textContent = `Domain Joined: ${entry.domain_joined ? 'Yes' : 'No'}${entry.ad_domain_formatted ? ` (${entry.ad_domain_formatted})` : ''}`;
		softwareInfoContainer.appendChild(domainJoined);
		const biosInfo = document.createElement('p');
		biosInfo.textContent = `BIOS: ${entry.bios_version || 'N/A'} ${entry.bios_updated ? '(Updated)' : '(Not Updated)'}`;
		softwareInfoContainer.appendChild(biosInfo);

		// Hardware info
		const hardwareInfoContainer = document.createElement('div');
		hardwareInfoContainer.classList.add('grid-item');
		const cpuUsage = document.createElement('p');
		cpuUsage.textContent = `CPU Usage: ${entry.cpu_usage !== null ? entry.cpu_usage.toFixed(2) + '%' : 'N/A'} ${entry.cpu_temp !== null ? `(` + entry.cpu_temp.toFixed(2) + '°C)' : ''}`;
		hardwareInfoContainer.appendChild(cpuUsage);
		const memoryUsage = document.createElement('p');
		memoryUsage.textContent = `Memory Usage: ${entry.memory_usage !== null && entry.memory_capacity !== null ? entry.memory_usage.toFixed(2) + 'GB / ' + entry.memory_capacity.toFixed(2) + 'GB' : 'N/A'}`;
		hardwareInfoContainer.appendChild(memoryUsage);
		const diskTemp = document.createElement('p');
		diskTemp.textContent = `Disk Temp: ${entry.disk_temp !== null ? entry.disk_temp.toFixed(2) + '°C' : 'N/A'}`;
		hardwareInfoContainer.appendChild(diskTemp);
		const networkUsage = document.createElement('p');
		networkUsage.textContent = `Network Usage: ${entry.network_usage !== null ? entry.network_usage.toFixed(2) + 'Mbps' : 'N/A'}`;
		hardwareInfoContainer.appendChild(networkUsage);
		const batteryCharge = document.createElement('p');
		batteryCharge.textContent = `Battery: ${entry.battery_charge !== null ? entry.battery_charge.toFixed(2) + '%' : 'N/A'} ${entry.battery_health ? '(Cap: ' + entry.battery_health?.toFixed(2) + '%' : 'N/A'} ${entry.battery_health_warning ? '(Warning)' : ''})`;
		hardwareInfoContainer.appendChild(batteryCharge);

		clientGridContainer.appendChild(clientIdentifiers);		
		if (entry.online) clientGridContainer.appendChild(liveViewContainer);
		clientGridContainer.appendChild(jobInfoContainer);
		if (entry.online) {
			onlineTableFragment.appendChild(clientEntryContainer);
		} else {
			offlineTableFragment.appendChild(clientEntryContainer);
		}
		clientGridContainer.appendChild(softwareInfoContainer);
		clientGridContainer.appendChild(hardwareInfoContainer);
	}
	onlineClientsDiv.innerHTML = '';
	onlineClientsDiv.appendChild(onlineTableFragment);
	offlineClientsDiv.innerHTML = '';
	offlineClientsDiv.appendChild(offlineTableFragment);
}

async function updateClientJob(tagnumber: number, job_name: string) {
	if (!tagnumber) {
		console.warn('tagnumber is null or undefined when trying to update client job');
		return;
	}

	if (!job_name) {
		console.warn('job_name is null or undefined when trying to update client job, defaulting to "cancel"');
		job_name = "cancel";
	}

	try {
		const response = await fetch('/api/job_queue/update_client_job', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({ tagnumber, job_name }),
		});
		if (!response.ok) {
			throw new Error('Server responded with ' + response.status);
		}
	} catch (error) {
		console.error('Error updating client job:', error);
		alert('An error occurred while updating the client job. Please try again.');
	}
}

async function initializeJobQueuePage() {
	const allJobs = await fetchAllJobs(true);
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
	const jobTable = await fetchJobQueueData();
	renderJobQueueTable(jobTable);
	if (jobQueueInterval) {
		clearInterval(jobQueueInterval);
	}
	jobQueueInterval = setInterval(async () => {
		const jobTable = await fetchJobQueueData();
		renderJobQueueTable(jobTable);
	}, 10000);
}

document.addEventListener('DOMContentLoaded', async () => {
	await initializeJobQueuePage();
});