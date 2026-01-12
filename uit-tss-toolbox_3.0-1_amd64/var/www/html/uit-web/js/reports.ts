// reports TS file

type ClientReport = {
	tagnumber: number;
	battery_health_pcnt: number;
	battery_health_stddev: number;
	battery_health_timestamp: Date;
}

const batteryHealthTbody = document.getElementById('battery-health-report-tbody');

async function populateBatteryStandardDeviationReport() {
	if (!batteryHealthTbody) { return; }

	try {
		const data = await fetchData(`/api/reports/battery/standard_deviation`, false);
		if (!data) { return; }
		data.sort((a: ClientReport, b: ClientReport) => {
			return (b.battery_health_stddev || 0) - (a.battery_health_stddev || 0);
		});

		for (const report of data) {
			const tr = document.createElement('tr');
			const tdTagNumber = document.createElement('td');
			const tagLink = document.createElement('a');
			tagLink.href = `/client/${report.tagnumber}`;
			tagLink.textContent = report.tagnumber?.toString() || 'N/A';
			tdTagNumber.appendChild(tagLink);
			tr.appendChild(tdTagNumber);

			const tdBatteryHealth = document.createElement('td');
			tdBatteryHealth.textContent = report.battery_health_pcnt !== undefined ? report.battery_health_pcnt.toFixed(2) + '%' : 'N/A';
			tr.appendChild(tdBatteryHealth);

			const tdBatteryStdDev = document.createElement('td');
			tdBatteryStdDev.textContent = report.battery_health_stddev !== undefined ? report.battery_health_stddev.toFixed(2) : 'N/A';
			tr.appendChild(tdBatteryStdDev);

			const tdTimestamp = document.createElement('td');
			tdTimestamp.textContent = report.battery_health_timestamp ? new Date(report.battery_health_timestamp).toLocaleString() : 'N/A';
			tr.appendChild(tdTimestamp);

			batteryHealthTbody.appendChild(tr);
		}
	} catch (error) {
		console.error('Error populating battery standard deviation report:', error);
	}
}

document.addEventListener('DOMContentLoaded', () => {
	populateBatteryStandardDeviationReport();
});
