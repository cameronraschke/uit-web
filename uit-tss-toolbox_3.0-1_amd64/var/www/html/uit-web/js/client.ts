const batteryContainer = document.getElementById('client-info-container');

type ClientInfoResponse = {
	Tagnumber:                 number | null
	SystemSerial:              string | null
	ClientUUID:                string | null
	LocationEntryTime:         Date | null
	Location:                  string | null
	Building:                  string | null
	Room:                      string | null
	DepartmentName:            string | null
	PropertyCustodian:         string | null
	AcquiredDate:              Date | null
	RetiredDate:               Date | null
	ClientStatus:              string | null
	IsBroken:                  boolean | null
	DiskRemoved:               boolean | null
	ClientNote:                string | null
	LocationLog:              any[] | null
	JobStartTime:                   Date | null
	CloneCompleted:            boolean | null
	CloneJobDuration:          number | null
	CloneImageName:            string | null
	EraseCompleted:            boolean | null
	EraseJobDuration:          number | null
	EraseMode:                 string | null
	JobLog:                    any[] | null
	IsCheckedOut:              boolean | null
	CheckoutDate:              Date | null
	ReturnDate:                Date | null
	CustomerName:              string | null
	CheckoutLog:               any[] | null
	FileCount:                 number | null
	ClientImages:              any[] | null
	LastOSEntryTime:           Date | null
	OSInstalled:               boolean | null
	OSName:                    string | null
	OSVersion:                 string | null
	ComputerName:              string | null
	OUName:                    string | null
	ADAdminUsers:              string[] | null
	IsIntuneJoined:            boolean | null
	IsBitlockerEnabled:        boolean | null
	LastHardwareCheck:         Date | null
	DiskHealthPcnt:            string | null
	BatteryHealthPcnt:         string | null
	DeviceType:                string | null
	BIOSVersion:               string | null
	BIOSReleaseDate:           string | null
	EthernetMAC:               string | null
	WiFiMAC:                   string | null
	TPMVersion:                string | null
	DiskModel:                 string | null
	DiskType:                  string | null
	DiskSizeKB:                number | null
	DiskSerial:                string | null
	DiskWritesKB:              number | null
	DiskReadsKB:               number | null
	DiskPowerOnHours:          number | null
	DiskErrors:                number | null
	DiskPowerCycles:           number | null
	DiskFirmware:              string | null
	BatteryManufacturer:       string | null
	BatteryModel:              string | null
	BatterySerial:             string | null
	BatteryManufactureDate:    string | null
	BatteryDesignCapacity:     number | null
	BatteryCurrentMaxCapacity: number | null
	BatteryChargeCycles:       number | null
	MemorySerial:              string | null
	MemoryCapacityKB:          number | null
	MemorySpeedMHz:            number | null
	SystemManufacturer:        string | null
	SystemModel:               string | null
	SystemSKU:                 string | null
	CPUManufacturer:           string | null
	CPUModel:                  string | null
	CPUMaxSpeedMhz:            number | null
	CPUCoreCount:              number | null
	CPUThreadCount:            number | null
}

async function fetchClientData(): Promise<any> {
	const path = '/api/client';
	const params = new URLSearchParams(window.location.search);
	const tagnumber = params.get('tagnumber');
	if (!tagnumber) {
		throw new Error('tagnumber parameter is missing or empty in URL');
	}
	const url = `${path}?tagnumber=${encodeURIComponent(tagnumber)}`;
	try {
		const response = await fetch(url);
		if (!response.ok) {
			throw new Error(`Error fetching client data: ${response.statusText}`);
		}
		const data: any = await response.json();
		return data;
	} catch (error) {
		console.error('Fetch client data failed:', error);
		return null;
	}
}

function renderClientData(data: ClientInfoResponse | null): void {
	if (!batteryContainer) {
		console.error('Battery container element not found');
		return;
	}

	if (!data) {
		console.error('No client data available');
		return;
	}

	const fragment = document.createDocumentFragment();
	const clientInfoEl = document.createElement('div');

	// Tag
	const tagEl = document.createElement('p');
	tagEl.textContent = `Tag: ${data.Tagnumber ?? 'N/A'}`;
	clientInfoEl.appendChild(tagEl);

	// System Serial
	const serialEl = document.createElement('p');
	serialEl.textContent = `System Serial: ${data.SystemSerial ?? 'N/A'}`;
	clientInfoEl.appendChild(serialEl);

	// Location
	const locationEl = document.createElement('p');
	locationEl.textContent = `Location: ${data.Location ?? 'N/A'}`;
	clientInfoEl.appendChild(locationEl);

	// Department
	const departmentEl = document.createElement('p');
	departmentEl.textContent = `Department: ${data.DepartmentName ?? 'N/A'}`;
	clientInfoEl.appendChild(departmentEl);

	// Client Status
	const statusEl = document.createElement('p');
	statusEl.textContent = `Status: ${data.ClientStatus ?? 'N/A'}`;
	clientInfoEl.appendChild(statusEl);

	fragment.appendChild(clientInfoEl);
	batteryContainer.appendChild(fragment);
}

document.addEventListener('DOMContentLoaded', async () => {
	const clientData: ClientInfoResponse = await fetchClientData();
	renderClientData(clientData);
});
