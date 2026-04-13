// Job Queue TypeScript File
type JobQueueTableRowView = {
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
	checkout_bool: boolean | null;
	kernel_updated: boolean | null;
	last_heard: Date | null;
	system_uptime: number | null;
	client_app_uptime: number | null;
	online: boolean | null;
	job_active: boolean | null;
	job_queued: boolean | null;
	job_queued_at: Date | null;
	job_queue_position: number | null;
	job_name: string | null;
	job_name_readable: string | null;
	job_clone_mode: string | null;
	job_erase_mode: string | null;
	job_status: string | null;
	last_job_time: Date | null;
	os_installed: boolean | null;
	os_name: string | null;
	os_updated: boolean | null;
	domain_joined: boolean | null;
	ad_domain: string | null;
	ad_domain_formatted: string | null;
	bios_updated: boolean | null;
	bios_version: string | null;
	cpu_current_usage: number; // Not null because cpu usage can be 0
	cpu_mhz: number | null;
	cpu_temp: number | null;
	cpu_temp_warning: boolean | null;
	memory_usage_kb: number | null;
	memory_capacity_kb: number | null;
	disk_usage: number | null;
	disk_temp: number | null;
	disk_type: string | null;
	disk_size_kb: number | null;
	max_disk_temp: number | null;
	disk_temp_warning: boolean | null;
	network_link_status: string | null;
	network_link_speed: number | null;
	network_usage: number | null;
	battery_charge_pcnt: number | null;
	battery_status: string | null;
	battery_health_deviation: number | null;
	battery_health_pcnt: number | null;
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
const onlineClientsCount = document.querySelector('#total-online-clients') as HTMLSpanElement | null;
const scrollToOnlineClientsButton = document.getElementById('jump-to-online-clients') as HTMLButtonElement;
const scrollToOfflineClientsButton = document.getElementById('jump-to-offline-clients') as HTMLButtonElement;

let jobQueueInterval: number | undefined;

scrollToOnlineClientsButton.addEventListener('click', () => {
	if (document.getElementById('online-clients-header') !== null) {
		document.getElementById('online-clients-header')!.scrollIntoView({ block: 'start', behavior: 'instant', inline: 'nearest' });
	}
});

scrollToOfflineClientsButton.addEventListener('click', () => {
	if (document.getElementById('offline-clients-header') !== null) {
		document.getElementById('offline-clients-header')!.scrollIntoView({ block: 'start', behavior: 'instant', inline: 'nearest' });
	}
});

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
		const data: AllJobs[] = await fetchData('/api/overview/job_queue/all_jobs', false);
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
		if (updateOnlineJobQueueSelect.value === '') {
			updateOnlineJobQueueButton.classList.add('disabled');
			updateOnlineJobQueueButton.disabled = true;
		} else {
			updateOnlineJobQueueButton.classList.remove('disabled');
			updateOnlineJobQueueButton.disabled = false;
		}
	});

	updateOnlineJobQueueForm.addEventListener('submit', async (event) => {
		event.preventDefault();
		updateOnlineJobQueueSelect.disabled = true;
		updateOnlineJobQueueButton.disabled = true;
		updateOnlineJobQueueSelect.classList.add('disabled');
		updateOnlineJobQueueButton.classList.add('disabled');

		const selectedValue = updateOnlineJobQueueSelect.value;
		if (!selectedValue) {
			alert('Please select a valid job to queue.');
			updateOnlineJobQueueSelect.classList.remove('disabled');
			updateOnlineJobQueueButton.classList.remove('disabled');
			updateOnlineJobQueueSelect.disabled = false;
			updateOnlineJobQueueButton.disabled = false;
			return;
		}

		try {
			const allJobsArr: AllJobs[] = await fetchAllJobs();
			if (!allJobsArr || !Array.isArray(allJobsArr) || allJobsArr.length === 0) {
				throw new Error('Failed to get available jobs from ' + '/api/overview/job_queue/all_jobs');
			}

			for (const job of allJobsArr) {
				if (job.job_name === selectedValue) {
					let clientJob: AllJobs = {
						job_name: job.job_name,
						job_name_readable: job.job_name_readable,
						job_sort_order: job.job_sort_order,
						job_hidden: job.job_hidden
					};
					const response = await fetch('/api/job_queue/all_clients/update_job', {
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
			updateOnlineJobQueueSelect.classList.remove('disabled');
			updateOnlineJobQueueButton.classList.remove('disabled');
			updateOnlineJobQueueSelect.disabled = false;
			updateOnlineJobQueueButton.disabled = false;
		}
	});
}

document.addEventListener('visibilitychange', async () => {
	if (jobQueueInterval) clearInterval(jobQueueInterval);
	if (document.visibilityState === 'visible') {
		// Fetch again
		const jobTable = await fetchJobQueueData();
		await renderJobQueueTable(jobTable);
		await startQueueInterval();
	}
});


async function fetchJobQueueData() : Promise<JobQueueTableRowView[] | []> {
	try {
		const data: JobQueueTableRowView[] = await fetchData('/api/overview/job_queue', false);
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



async function renderJobQueueTable(data: JobQueueTableRowView[]) {
	if (!data || !Array.isArray(data) || data.length === 0) {
		console.warn('No job queue data to render.');
		return;
	}
	if (!onlineClientsDiv || !offlineClientsDiv) return;

	// Sort array
	data.sort((a, b) => {
		if (!a.online && !b.online && a.last_heard !== b.last_heard) {
			if (a.last_heard === null) return 1;
			if (b.last_heard === null) return -1;
			return new Date(b.last_heard).getTime() - new Date(a.last_heard).getTime();
		}

		if (a.online !== b.online) {
			return (a.online === true) ? -1 : 1;
		}

		if (a.job_active !== b.job_active) {
			return (a.job_active === true) ? -1 : 1;
		}

		if (a.job_queued !== b.job_queued) {
			return (a.job_queued === true) ? -1 : 1;
		}

		if (a.job_queue_position !== b.job_queue_position) {
			if (a.job_queue_position === null) return 1;
			if (b.job_queue_position === null) return -1;
			return a.job_queue_position - b.job_queue_position;
		}

		if (a.system_uptime !== b.system_uptime) {
			if (a.system_uptime === null) return 1;
			if (b.system_uptime === null) return -1;
			return a.system_uptime - b.system_uptime;
		}

		if (a.client_app_uptime !== b.client_app_uptime) {
			if (a.client_app_uptime === null) return 1;
			if (b.client_app_uptime === null) return -1;
			return b.client_app_uptime - a.client_app_uptime;
		}

		return (a.tagnumber ?? 0) - (b.tagnumber ?? 0);
	});

	let jobs: AllJobs[] = [];
	try {
		jobs = await fetchAllJobs();
	} catch (error) {
		console.error('Error fetching jobs for table render:', error);
	}

	const onlineTableFragment = document.createDocumentFragment();
	const offlineTableFragment = document.createDocumentFragment();

	let totalOnlineClients = 0;
	for (const entry of data) {
		if (entry && entry.online !== null && entry.online) {
			totalOnlineClients++;
		}
		if (!entry.tagnumber) {
			console.warn("No tagnumber for entry");
			continue;
		}

		const clientEntryContainer = document.createElement('div');
		clientEntryContainer.classList.add('client-row-container');
		clientEntryContainer.dataset.tagnumber = entry.tagnumber.toString();

		if (entry.is_broken) {
			clientEntryContainer.classList.add('broken');
		}
		if (entry.online) {
			clientEntryContainer.classList.add('online');
		} else {
			clientEntryContainer.classList.add('offline');
		}

		const clientGridContainer = document.createElement('div');
		clientGridContainer.classList.add('client-row-grid');

		// Client identifiers and info
		const clientIdentifiers = document.createElement('div');
		clientIdentifiers.className = 'grid-item headers';
		if (entry.online === false || entry.is_broken === true) {
			const brokenClient = document.createElement('p');
			brokenClient.classList.add('bold-text');
			if (!entry.online) {
				brokenClient.appendChild(document.createTextNode(' [Offline]'));
			}
			if (entry.is_broken) {
				brokenClient.appendChild(document.createTextNode('[Broken Client]'));
			}
			clientIdentifiers.appendChild(brokenClient);
		}

		const tagAndSerialDiv = document.createElement('div');
		tagAndSerialDiv.classList.add('flex-container', 'horizontal');
		tagAndSerialDiv.style.justifyContent = 'center';

		// tagAndSerialDiv.appendChild(editClientAnchor);

		const tagSerialContainer = document.createElement('p');
		// const tagURL = new URL(`/client`, window.location.origin);
		const tagURL = new URL(`/inventory`, window.location.origin);
		tagURL.searchParams.append('tagnumber', entry.tagnumber.toString());
		tagURL.searchParams.append('system_serial', entry.system_serial || '');
		tagURL.searchParams.append('update', 'true');
		const editClientAnchor = document.createElement('a');
		editClientAnchor.title = `Edit Client ${entry.tagnumber !== null ? entry.tagnumber.toString() : 'N/A'}`;
		editClientAnchor.textContent = `${entry.tagnumber !== null ? entry.tagnumber.toString() : 'N/A'}`;
		editClientAnchor.href = tagURL.toString();
		editClientAnchor.target = '_blank';
		
		const editClientSvg = document.createElement('img');
		editClientSvg.src = '/icons/general/edit_square.svg';
		editClientAnchor.appendChild(editClientSvg);
		editClientSvg.style.marginRight = '0.5rem';
		editClientAnchor.appendChild(editClientSvg);
		tagSerialContainer.appendChild(editClientAnchor);

		const tagTextNode = document.createElement('span').appendChild(editClientAnchor);
		const serialTextNode = document.createElement('span').appendChild(document.createTextNode(entry.system_serial ? ` | ${entry.system_serial}` : ' | N/A'));
		tagSerialContainer.appendChild(tagTextNode);
		tagSerialContainer.appendChild(document.createElement('wbr'));
		tagSerialContainer.appendChild(serialTextNode);
		tagAndSerialDiv.appendChild(tagSerialContainer);

		clientIdentifiers.appendChild(tagAndSerialDiv);

		const manufacturerModelContainer = document.createElement('div');
		manufacturerModelContainer.classList.add('flex-container', 'horizontal');
		manufacturerModelContainer.style.justifyContent = 'center';
		manufacturerModelContainer.style.marginBottom = '1rem';
		if (entry.system_manufacturer) {
			manufacturerModelContainer.appendChild(document.createElement('span').appendChild(document.createTextNode(entry.system_manufacturer)));
		} else {
			manufacturerModelContainer.appendChild(document.createElement('span').appendChild(document.createTextNode('N/A')));
		}
		manufacturerModelContainer.appendChild(document.createElement('span').appendChild(document.createTextNode(' | ')));
		manufacturerModelContainer.appendChild(document.createElement('wbr'));
		if (entry.system_model) {
			manufacturerModelContainer.appendChild(document.createElement('span').appendChild(document.createTextNode(entry.system_model)));
		} else {
			manufacturerModelContainer.appendChild(document.createElement('span').appendChild(document.createTextNode('N/A')));
		}
		clientIdentifiers.appendChild(manufacturerModelContainer);

		const locationContainer = document.createElement('div');
		locationContainer.classList.add('flex-container', 'horizontal');
		locationContainer.classList.add('smaller-text');
		locationContainer.appendChild(document.createElement('span').appendChild(document.createTextNode('Location: ')));
		if (entry.location !== null) {
			locationContainer.appendChild(document.createTextNode(entry.location));
		} else {
			locationContainer.appendChild(document.createTextNode('N/A'));
		}
		clientIdentifiers.appendChild(locationContainer);

		const departmentContainer = document.createElement('div');
		departmentContainer.classList.add('flex-container', 'horizontal');
		departmentContainer.classList.add('smaller-text');
		departmentContainer.appendChild(document.createElement('span').appendChild(document.createTextNode('Department: ')));
		if (entry.department_name !== null) {
			departmentContainer.appendChild(document.createTextNode(entry.department_name));
		} else {
			departmentContainer.appendChild(document.createTextNode('N/A'));
		}
		clientIdentifiers.appendChild(departmentContainer);

		const clientStatusContainer = document.createElement('div');
		clientStatusContainer.classList.add('flex-container', 'horizontal');
		clientStatusContainer.classList.add('smaller-text');
		clientStatusContainer.appendChild(document.createElement('span').appendChild(document.createTextNode('Client Status: ')));
		if (entry.client_status !== null) {
			clientStatusContainer.appendChild(document.createTextNode(entry.client_status));
		} else {
			clientStatusContainer.appendChild(document.createTextNode('N/A'));
		}
		clientIdentifiers.appendChild(clientStatusContainer);

		// Live view
		const liveViewContainer = document.createElement('div');
		liveViewContainer.classList.add('grid-item');

		const liveViewHeader = document.createElement('p');
		liveViewHeader.style.fontStyle = 'italic';
		liveViewHeader.textContent = 'Live View: ';
		liveViewContainer.appendChild(liveViewHeader);

		const liveViewScreenshotContainer = document.createElement('div');
		liveViewScreenshotContainer.classList.add('image-container');
		if (entry.online) {
			liveViewScreenshotContainer.classList.add('image-container');
			liveViewScreenshotContainer.classList.remove('offline');
			const liveViewImage = document.createElement('img');
			liveViewImage.src = `/api/live_image?tagnumber=${entry.tagnumber}`;
			liveViewImage.loading = "lazy";
			const liveViewAnchor = document.createElement('a');
			liveViewAnchor.href = `/api/live_image?tagnumber=${entry.tagnumber}`;
			liveViewAnchor.target = "_blank";
			liveViewAnchor.appendChild(liveViewImage);
			liveViewScreenshotContainer.appendChild(liveViewAnchor);
		}

		liveViewContainer.appendChild(liveViewScreenshotContainer);
		clientEntryContainer.appendChild(clientGridContainer);

		// Job info
		const jobInfoContainer = document.createElement('div');
		jobInfoContainer.classList.add('grid-item');

		if (entry.online === false || (entry.job_name_readable !== null && entry.job_queued !== null && entry.job_queue_position !== null)) {
			const jobStatusP = document.createElement('p');
			jobStatusP.appendChild(document.createTextNode(`Job Status: ${entry.job_status || 'N/A'}`));
			jobInfoContainer.appendChild(jobStatusP);
			const jobNameP = document.createElement('p');
			jobNameP.appendChild(document.createTextNode(`Job Name: ${entry.job_name_readable || 'N/A'}`));
			jobInfoContainer.appendChild(jobNameP);
			const jobQueuePositionP = document.createElement('p');
			jobQueuePositionP.appendChild(document.createTextNode(`Queue Position: ${entry.job_queue_position || 'N/A'}`));
			jobInfoContainer.appendChild(jobQueuePositionP);
		}

		// Uptime
		const clientAppUptimeContainer = document.createElement('p');
		clientAppUptimeContainer.appendChild(document.createTextNode('App Uptime: '));
		if (entry.client_app_uptime !== null) {
			const uptimeSec = entry.client_app_uptime || 0;
			const uptimeMins = Math.floor(uptimeSec / 60) || 0;
			const uptimeHours = Math.floor(uptimeMins / 60) || 0;
			const uptimeDays = Math.floor(uptimeHours / 24) || 0;
			if (uptimeDays > 0) {
				const span = document.createElement('span');
				span.appendChild(document.createTextNode(`${uptimeDays}d ${uptimeHours % 24}h ${uptimeMins % 60}m`));
				clientAppUptimeContainer.appendChild(span);
			} else if (uptimeHours > 0) {
				const span = document.createElement('span');
				span.appendChild(document.createTextNode(`${uptimeHours}h ${uptimeMins % 60}m`));
				clientAppUptimeContainer.appendChild(span);
			} else if (uptimeMins > 0) {
				const span = document.createElement('span');
				span.appendChild(document.createTextNode(`${uptimeMins}m ${uptimeSec % 60}s`));
				clientAppUptimeContainer.appendChild(span);
			} else if (uptimeSec > 0) {
				const span = document.createElement('span');
				span.appendChild(document.createTextNode(`${uptimeSec}s`));
				clientAppUptimeContainer.appendChild(span);
			} else {
				clientAppUptimeContainer.appendChild(document.createTextNode('N/A'));
			}
		} else {
			clientAppUptimeContainer.appendChild(document.createTextNode('N/A'));
		}
		jobInfoContainer.appendChild(clientAppUptimeContainer);

		const systemUptimeContainer = document.createElement('p');
		systemUptimeContainer.appendChild(document.createTextNode('System Uptime: '));
		if (entry.system_uptime !== null) {
			const uptimeSec = entry.system_uptime || 0;
			const uptimeMins = Math.floor(uptimeSec / 60) || 0;
			const uptimeHours = Math.floor(uptimeMins / 60) || 0;
			const uptimeDays = Math.floor(uptimeHours / 24) || 0;
			if (uptimeDays > 0) {
				const span = document.createElement('span');
				span.appendChild(document.createTextNode(`${uptimeDays}d ${uptimeHours % 24}h ${uptimeMins % 60}m`));
				systemUptimeContainer.appendChild(span);
			} else if (uptimeHours > 0) {
				const span = document.createElement('span');
				span.appendChild(document.createTextNode(`${uptimeHours}h ${uptimeMins % 60}m`));
				systemUptimeContainer.appendChild(span);
			} else if (uptimeMins > 0) {
				const span = document.createElement('span');
				span.appendChild(document.createTextNode(`${uptimeMins}m ${uptimeSec % 60}s`));
				systemUptimeContainer.appendChild(span);
			} else if (uptimeSec > 0) {
				const span = document.createElement('span');
				span.appendChild(document.createTextNode(`${uptimeSec}s`));
				systemUptimeContainer.appendChild(span);
			} else {
				systemUptimeContainer.appendChild(document.createTextNode('N/A'));
			} 
		} else {
			systemUptimeContainer.appendChild(document.createTextNode('N/A'));
		}

		jobInfoContainer.appendChild(systemUptimeContainer);

		const clientActionsContainer = document.createElement('div');
		clientActionsContainer.classList.add('flex-container', 'horizontal');
		clientActionsContainer.style.justifyContent = 'flex-start';

		const jobSelectContainer = document.createElement('div');
		jobSelectContainer.classList.add('flex-container', 'horizontal');
		const jobSelect = document.createElement('select');
		const existingSelectElID = `${entry.tagnumber}-job-select`;
		const existingSelectEl = document.getElementById(existingSelectElID) as HTMLSelectElement;
		
		jobSelect.id = existingSelectElID
		
		const defaultOption = document.createElement('option');
		defaultOption.value = existingSelectEl && existingSelectEl.options[existingSelectEl.selectedIndex].value ? existingSelectEl.options[existingSelectEl.selectedIndex].value : '';
		defaultOption.textContent = existingSelectEl && existingSelectEl.options[existingSelectEl.selectedIndex].text ? existingSelectEl.options[existingSelectEl.selectedIndex].text : 'Select job to queue';
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

		jobSelect.addEventListener('focus', () => {
			if (jobQueueInterval) {
				clearInterval(jobQueueInterval);
				jobQueueInterval = undefined;
			}
		});

		jobSelect.addEventListener('click', () => {
			if (jobQueueInterval) {
				clearInterval(jobQueueInterval);
				jobQueueInterval = undefined;
			}
		});

		jobSelect.addEventListener('blur', async () => {
			await startQueueInterval();
		});
			
		
		clientActionsContainer.appendChild(jobSelect);
		
		const queueJobButton = document.createElement('button');
		queueJobButton.removeEventListener('click', async () => {});
		if (entry.job_queued) {
			queueJobButton.textContent = 'Cancel Job';
			queueJobButton.classList.add('svg-button', 'cancel');
			jobSelect.value = entry.job_name || '';
			jobSelect.disabled = true;
			jobSelect.classList.add('disabled');
			if (entry.job_name === "cancel" || entry.job_name === "shutdown") {
				queueJobButton.disabled = true;
				queueJobButton.classList.add('disabled');
			} else {
				queueJobButton.addEventListener('click', async () => {
					try {
							if (entry.tagnumber === null) {
								throw new Error('tagnumber is null');
							}
							await updateClientJob(entry.tagnumber, "cancel");
						} catch (error) {
							console.error('Error canceling job:', error);
							alert('An error occurred while canceling the job. Please try again.');
						} finally {
							await initializeJobQueuePage();
						}
					});
			}
		} else {
			queueJobButton.textContent = 'Queue Job';
			queueJobButton.classList.remove('svg-button', 'cancel');
			jobSelect.disabled = false;
			jobSelect.classList.remove('disabled');
			queueJobButton.addEventListener('click', async () => {
				if (!jobSelect.value) {
					alert('Please select a job to queue.');
					return;
				}
					
				try {
					if (entry.tagnumber === null || !jobSelect.value) {
						throw new Error('tagnumber or selected job is null');
					}
					await updateClientJob(entry.tagnumber, jobSelect.value);
				} catch (error) {
					console.error('Error queueing job:', error);
					alert('An error occurred while queueing the job. Please try again.');
				} finally {
					await initializeJobQueuePage();
				}
			});
		}
		clientActionsContainer.appendChild(queueJobButton);

		if (entry.online) jobInfoContainer.appendChild(clientActionsContainer);

		if (!entry.online) {
			const lastHeard = document.createElement('p');
			lastHeard.textContent = `Last Heard: ${entry.last_heard ? new Date(entry.last_heard).toLocaleString() : 'N/A'}`;
			jobInfoContainer.appendChild(lastHeard);
		}

		// Software info
		const softwareInfoContainer = document.createElement('div');
		softwareInfoContainer.classList.add('grid-item');
		const osInfo = document.createElement('p');
		osInfo.appendChild(document.createTextNode('OS: '));
		if (entry.os_name) {
			const osNameSpan = document.createElement('span');
			osNameSpan.appendChild(document.createTextNode(entry.os_name));
			osInfo.appendChild(osNameSpan);
		} else {
			const osNameSpan = document.createElement('span');
			osNameSpan.appendChild(document.createTextNode('N/A'));
			osNameSpan.style.fontStyle = 'italic';
			osInfo.appendChild(osNameSpan);
		}
		if (entry.os_updated !== null && entry.os_updated === true) {
			const osWarning = document.createElement('span');
			osWarning.style.color = "green";
			osWarning.appendChild(document.createTextNode(' (Updated)'));
			osInfo.appendChild(osWarning);
		} else {
			const osWarning = document.createElement('span');
			osWarning.style.color = "red";
			osWarning.appendChild(document.createTextNode(' (Out of date)'));
			osInfo.appendChild(osWarning);
		}
		softwareInfoContainer.appendChild(osInfo);
		const domainJoined = document.createElement('p');
		domainJoined.appendChild(document.createTextNode('AD Domain: '));
		if (entry.domain_joined === true && entry.ad_domain_formatted !== null) {
			domainJoined.appendChild(document.createTextNode(entry.ad_domain_formatted))
		} else {
			domainJoined.appendChild(document.createTextNode('No'));
		}
		softwareInfoContainer.appendChild(domainJoined);

		softwareInfoContainer.appendChild(document.createElement('span').appendChild(document.createTextNode(' Last Job: ')));
		if (entry.last_job_time) {
			const lastJobTime = document.createElement('span');
			if (entry.os_installed === true) {
				lastJobTime.appendChild(document.createTextNode('Clone - '));
			} if (entry.os_installed === false) {
				lastJobTime.appendChild(document.createTextNode('Erase - '));
			}
			lastJobTime.appendChild(document.createTextNode(new Date(entry.last_job_time).toLocaleString()));
			softwareInfoContainer.appendChild(lastJobTime);
		} else {
			const lastJobTime = document.createElement('span');
			lastJobTime.appendChild(document.createTextNode('N/A'));
			lastJobTime.style.fontStyle = 'italic';
			softwareInfoContainer.appendChild(lastJobTime);
		}

		const biosInfo = document.createElement('p');
		const biosText = document.createTextNode('BIOS: ');
		biosInfo.appendChild(biosText);
		if (entry.bios_version !== null) {
			biosInfo.appendChild(document.createTextNode(entry.bios_version));
			if (entry.bios_updated) {
				const biosWarning = document.createElement('span');
				biosWarning.style.color = "green";
				biosWarning.appendChild(document.createTextNode(' (Updated)'));
				biosInfo.appendChild(biosWarning);
			} else {
				const biosWarning = document.createElement('span');
				biosWarning.style.color = "red";
				biosWarning.appendChild(document.createTextNode(' (Out of date)'));
				biosInfo.appendChild(biosWarning);
			}
		} else {
			biosInfo.appendChild(document.createTextNode('N/A'));
		}
		softwareInfoContainer.appendChild(biosInfo);

		// Hardware info
		const hardwareInfoContainer = document.createElement('div');
		hardwareInfoContainer.classList.add('grid-item');

		// CPU
		const cpuUsage = document.createElement('p');
		cpuUsage.appendChild(document.createTextNode('CPU: '));
		cpuUsage.appendChild(document.createElement('wbr'));
		// CPU Usage
		if (entry.cpu_current_usage !== null && entry.cpu_current_usage >= 0) {
			cpuUsage.appendChild(document.createTextNode(`${entry.cpu_current_usage.toFixed(2)}` + '%'))
		} else {
			cpuUsage.appendChild(document.createTextNode('N/A'));
		}
		cpuUsage.appendChild(document.createElement('wbr'));
		// CPU MHz (frequency)
		if (entry.cpu_mhz !== null) {
			const cpuMHz = document.createElement('span');
			cpuMHz.appendChild(document.createTextNode(' @' + `${(entry.cpu_mhz / 1000).toFixed(2)}` + 'GHz'));
			cpuUsage.appendChild(cpuMHz);
			cpuUsage.appendChild(document.createElement('wbr'));
		}
		// CPU Temp
		if (entry.cpu_temp !== null) {
			cpuUsage.appendChild(document.createTextNode(' ('));
			const cpuTemp = document.createElement('span');
			if (entry.cpu_temp_warning) {
				cpuTemp.style.color = 'red';
				cpuTemp.classList.add('bold-text');
			}
			cpuTemp.appendChild(document.createTextNode(`${entry.cpu_temp.toFixed(0)}` + '°C'));
			cpuUsage.appendChild(cpuTemp);
			cpuUsage.appendChild(document.createTextNode(')'));
		} else {
			cpuUsage.appendChild(document.createTextNode(' (Temp N/A)'));
		}
		cpuUsage.appendChild(document.createElement('wbr'));
		hardwareInfoContainer.appendChild(cpuUsage);

		// Memory/RAM
		const memoryUsage = document.createElement('p');
		memoryUsage.appendChild(document.createTextNode('Memory: '));
		memoryUsage.appendChild(document.createElement('wbr'));
		if (entry.memory_usage_kb !== null && entry.memory_capacity_kb !== null) {
			memoryUsage.appendChild(document.createTextNode(`${(entry.memory_usage_kb / 1024 / 1024).toFixed(2)}` + 'GB / ' + `${(entry.memory_capacity_kb / 1024 / 1024).toFixed(2)}` + 'GB'));
		} else {
			memoryUsage.appendChild(document.createTextNode('N/A'));
		}
		hardwareInfoContainer.appendChild(memoryUsage);

		// Disk
		const diskTemp = document.createElement('p');
		diskTemp.appendChild(document.createTextNode('Disk Temp: '));
		diskTemp.appendChild(document.createElement('wbr'));
		if (entry.disk_temp !== null) {
			const diskTempValue = document.createElement('span');
			if (entry.disk_temp_warning) {
				diskTempValue.style.color = 'red';
			}
			diskTempValue.appendChild(document.createTextNode(`${entry.disk_temp.toFixed(0)}` + '°C'));
			diskTemp.appendChild(diskTempValue);
		} else {
			diskTemp.appendChild(document.createTextNode('N/A'));
		}
		hardwareInfoContainer.appendChild(diskTemp);

		// Network
		const networkUsage = document.createElement('p');
		networkUsage.appendChild(document.createTextNode('Network Usage: '));
		networkUsage.appendChild(document.createElement('wbr'));
		if (entry.network_usage !== null) {
			const networkUsageValue = document.createElement('span');
			networkUsageValue.appendChild(document.createTextNode(`${entry.network_usage.toFixed(2)}` + 'Mbps'));
			networkUsage.appendChild(networkUsageValue);
		} else {
			networkUsage.appendChild(document.createTextNode('N/A'));
		}
		hardwareInfoContainer.appendChild(networkUsage);

		// Battery
		const batteryInfo = document.createElement('p');
		const batteryInfoText = document.createTextNode('Battery: ');
		if (entry.battery_charge_pcnt !== null) batteryInfoText.appendData(`${entry.battery_charge_pcnt.toFixed(0)}%`); else batteryInfoText.appendData('N/A%');
		if (entry.battery_health_pcnt !== null) batteryInfoText.appendData(` (Max Capacity: ${entry.battery_health_pcnt?.toFixed(2)}%`);
		if (entry.battery_health_deviation !== null) {
			if (entry.battery_health_deviation) batteryInfoText.appendData(', ');
			if (entry.battery_health_deviation > 0) {
				if (entry.battery_health_deviation) batteryInfoText.appendData(`+${entry.battery_health_deviation.toFixed(2)}`);
			} else if (entry.battery_health_deviation < 0) {
				if (entry.battery_health_deviation) batteryInfoText.appendData(`${entry.battery_health_deviation.toFixed(2)}`);
			}
		}
		batteryInfoText.appendData(')');
		batteryInfo.appendChild(batteryInfoText);
		hardwareInfoContainer.appendChild(batteryInfo);

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
	if (onlineClientsCount) onlineClientsCount.textContent = totalOnlineClients.toString() || '0';
	onlineClientsDiv.innerHTML = '';
	onlineClientsDiv.appendChild(onlineTableFragment);

	const separationDiv = document.createElement('div');
	separationDiv.classList.add('separation-div');
	separationDiv.appendChild(document.createElement('hr'));
	onlineClientsDiv.appendChild(separationDiv);

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
		const response = await fetch('/api/client/job_queue/update_job', {
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

async function startQueueInterval() {
	// Clear interval
	if (jobQueueInterval) clearInterval(jobQueueInterval);

	// Restart interval
	jobQueueInterval = setInterval(async () => {
		const jobTable = await fetchJobQueueData();
		if (!jobQueueInterval) return;
		await renderJobQueueTable(jobTable);
	}, 10000);
}

async function initializeJobQueuePage() {
	const allJobs = await fetchAllJobs(true);
	
	if (updateOnlineJobQueueSelect) {
		updateOnlineJobQueueSelect.addEventListener('focus', () => {
			if (jobQueueInterval) {
				clearInterval(jobQueueInterval);
				jobQueueInterval = undefined;
			}
		});
		updateOnlineJobQueueSelect.addEventListener('click', () => {
			if (jobQueueInterval) {
				clearInterval(jobQueueInterval);
				jobQueueInterval = undefined;
			}
		});
		updateOnlineJobQueueSelect.addEventListener('blur', async () => {
			await startQueueInterval();
		});

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

	// Initial fetch and render of job queue table
	const jobTable = await fetchJobQueueData();
	await renderJobQueueTable(jobTable);
	await startQueueInterval();
}

document.addEventListener('DOMContentLoaded', async () => {
	await initializeJobQueuePage();
});