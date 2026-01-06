const batteryContainer = document.getElementById('battery-container');

type BatteryHealth = {
	time: string | null;
	tagnumber: number | null;
	jobstatsHealthPcnt: number | null;
	clientHealthPcnt: number | null;
	chargeCycles: number | null;
};

async function fetchBatteryHealthData(): Promise<BatteryHealth[]> {
	const url = '/api/client/health/battery';
	const params = new URLSearchParams(window.location.search);
	if (params.has('tagnumber')) {
		url.concat(`?tagnumber=${params.get('tagnumber')}`);
	}
	try {
		const response = await fetch(url);
		if (!response.ok) {
			throw new Error(`Error fetching battery health data: ${response.statusText}`);
		}
		const data: BatteryHealth[] = await response.json();
		return data;
	} catch (error) {
		console.error('Fetch battery health data failed:', error);
		return [];
	}
}

function renderBatteryHealth(data: BatteryHealth[]) {
	if (!batteryContainer) return;
	batteryContainer.innerHTML = '';

	if (data.length === 0) {
		const message = document.createElement('p');
		message.textContent = 'No battery health data available.';
		batteryContainer.appendChild(message);
		return;
	}
	const batteryTable = document.createElement('table');
	const thead = document.createElement('thead');
	const headerRow = document.createElement('tr');
	const headers = ['Tag Number', 'Jobstats Health (%)', 'Client Health (%)', 'Charge Cycles', 'Last Updated'];
	for (const headerText of headers) {
		const th = document.createElement('th');
		th.textContent = headerText;
		headerRow.appendChild(th);
	}
	thead.appendChild(headerRow);
	batteryTable.appendChild(thead);
	const batteryTbody = document.createElement('tbody');
	for (const row of data) {
		const tr = document.createElement('tr');
		const timeCell = document.createElement('td');
		timeCell.textContent = row.time !== null ? new Date(row.time).toLocaleString() : 'N/A';
		tr.appendChild(timeCell);
		batteryTbody.appendChild(tr);
		const tagCell = document.createElement('td');
		tagCell.textContent = row.tagnumber !== null ? row.tagnumber.toString() : 'N/A';
		tr.appendChild(tagCell);
		const jobstatsHealthCell = document.createElement('td');
		jobstatsHealthCell.textContent = row.jobstatsHealthPcnt !== null ? row.jobstatsHealthPcnt.toString() : 'N/A';
		tr.appendChild(jobstatsHealthCell);
		const clientHealthCell = document.createElement('td');
		clientHealthCell.textContent = row.clientHealthPcnt !== null ? row.clientHealthPcnt.toString() : 'N/A';
		tr.appendChild(clientHealthCell);
		const chargeCyclesCell = document.createElement('td');
		chargeCyclesCell.textContent = row.chargeCycles !== null ? row.chargeCycles.toString() : 'N/A';
		tr.appendChild(chargeCyclesCell);
	}
	batteryTable.appendChild(batteryTbody);
	batteryContainer.appendChild(batteryTable);
}

document.addEventListener('DOMContentLoaded', async () => {
	const batteryData = await fetchBatteryHealthData();
	renderBatteryHealth(batteryData);
});
