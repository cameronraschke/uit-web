const clientInfoContainer = document.getElementById('client-info-container') as HTMLDivElement | null;
const pageTitle = document.getElementById('page-title') as HTMLHeadingElement | null;

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
	IsDiskEncrypted:        boolean | null
	LastHardwareCheck:         Date | null
	DiskHealthPcnt:            number | null
	BatteryHealthPcnt:         number | null
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

async function fetchClientData(): Promise<ClientInfoResponse | null> {
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
		const data: ClientInfoResponse = await response.json();
		return data;
	} catch (error) {
		console.error('Fetch client data failed:', error);
		return null;
	}
}

function renderClientData(data: ClientInfoResponse | null): void {
	if (!clientInfoContainer || !pageTitle) {
		console.error('Client info container or page title element not found');
		return;
	}

	const tag = new URLSearchParams(window.location.search).get('tagnumber');
	document.title = `${tag} - Client Details - UIT Toolbox`;
	pageTitle.textContent = `Client Details - ${tag ?? 'N/A'}`;
	if (!data) {
		console.error('No client data available');
		return;
	}

	const fragment = document.createDocumentFragment();
	const clientIDsDiv = document.createElement('div');

	// Tag
	const tagEl = document.createElement('p');
	tagEl.textContent = `Tag: ${data.Tagnumber ?? 'N/A'}`;
	clientIDsDiv.appendChild(tagEl);

	// System Serial
	const serialEl = document.createElement('p');
	serialEl.textContent = `System Serial: ${data.SystemSerial ?? 'N/A'}`;
	clientIDsDiv.appendChild(serialEl);

	// Client UUID
	const ClientUUIDEl = document.createElement('p');
	ClientUUIDEl.textContent = `Client UUID: ${data.ClientUUID ?? 'N/A'}`;
	clientIDsDiv.appendChild(ClientUUIDEl);
	
	fragment.appendChild(clientIDsDiv);

	// Location, department, property, status
	const locationInfoDiv = document.createElement('div');

	// Location
	const locationEl = document.createElement('p');
	locationEl.textContent = `Location: ${data.Location ?? 'N/A'}`;
	locationInfoDiv.appendChild(locationEl);

	// Building
	const buildingEl = document.createElement('p');
	buildingEl.textContent = `Building: ${data.Building ?? 'N/A'}`;
	locationInfoDiv.appendChild(buildingEl);

	// Room
	const roomEl = document.createElement('p');
	roomEl.textContent = `Room: ${data.Room ?? 'N/A'}`;
	locationInfoDiv.appendChild(roomEl);

	// Department
	const departmentEl = document.createElement('p');
	departmentEl.textContent = `Department: ${data.DepartmentName ?? 'N/A'}`;
	locationInfoDiv.appendChild(departmentEl);

	fragment.appendChild(locationInfoDiv);


	// Client Status
	const clientStatusDiv = document.createElement('div');
	const statusEl = document.createElement('p');
	statusEl.textContent = `Status: ${data.ClientStatus ?? 'N/A'}`;
	clientStatusDiv.appendChild(statusEl);

	// note 
	if (data.ClientNote) {
		const noteEl = document.createElement('p');
		noteEl.textContent = `Note: ${data.ClientNote}`;
		clientStatusDiv.appendChild(noteEl);
	}

	fragment.appendChild(clientStatusDiv);



	// Property info
	const propertyInfoDiv = document.createElement('div');

	// Property Custodian
	const custodianEl = document.createElement('p');
	custodianEl.textContent = `Property Custodian: ${data.PropertyCustodian ?? 'N/A'}`;
	propertyInfoDiv.appendChild(custodianEl);

	// Acquired Date
	const acquiredDateEl = document.createElement('p');
	acquiredDateEl.textContent = `Acquired Date: ${data.AcquiredDate ? new Date(data.AcquiredDate).toLocaleDateString() : 'N/A'}`;
	propertyInfoDiv.appendChild(acquiredDateEl);

	// Retired Date
	const retiredDateEl = document.createElement('p');
	retiredDateEl.textContent = `Retired Date: ${data.RetiredDate ? new Date(data.RetiredDate).toLocaleDateString() : 'N/A'}`;
	propertyInfoDiv.appendChild(retiredDateEl);

	// Disk Removed
	const diskRemovedEl = document.createElement('p');
	diskRemovedEl.textContent = `Disk Removed: ${data.DiskRemoved !== null ? (data.DiskRemoved ? 'Yes' : 'No') : 'N/A'}`;
	propertyInfoDiv.appendChild(diskRemovedEl);

	fragment.appendChild(propertyInfoDiv);


	// OS Info
	const osInfoDiv = document.createElement('div');

	// Last OS Entry Time
	const lastOsEntryEl = document.createElement('p');
	lastOsEntryEl.textContent = `Last OS Entry Time: ${data.LastOSEntryTime ? new Date(data.LastOSEntryTime).toLocaleDateString() : 'N/A'}`;
	osInfoDiv.appendChild(lastOsEntryEl);

	// OS Name
	const osNameEl = document.createElement('p');
	osNameEl.textContent = `OS Name: ${data.OSName ?? 'N/A'}`;
	osInfoDiv.appendChild(osNameEl);

	// OS Version
	const osVersionEl = document.createElement('p');
	osVersionEl.textContent = `OS Version: ${data.OSVersion ?? 'N/A'}`;
	osInfoDiv.appendChild(osVersionEl);

	// AD Domain / OU
	const ouEl = document.createElement('p');
	ouEl.textContent = `AD OU: ${data.OUName ?? 'N/A'}`;
	osInfoDiv.appendChild(ouEl);

	// Computer Name
	const computerNameEl = document.createElement('p');
	computerNameEl.textContent = `Computer Name: ${data.ComputerName ?? 'N/A'}`;
	osInfoDiv.appendChild(computerNameEl);

	fragment.appendChild(osInfoDiv);


	// Client Health
	const healthInfoDiv = document.createElement('div');

	// Last Hardware Check
	const lastHardwareCheckEl = document.createElement('p');
	lastHardwareCheckEl.textContent = `Last Hardware Check: ${data.LastHardwareCheck ? new Date(data.LastHardwareCheck).toLocaleDateString() : 'N/A'}`;
	healthInfoDiv.appendChild(lastHardwareCheckEl);

	// Is Broken
	const isBrokenEl = document.createElement('p');
	isBrokenEl.textContent = `Is Broken: ${data.IsBroken !== null ? (data.IsBroken ? 'Yes' : 'No') : 'N/A'}`;
	healthInfoDiv.appendChild(isBrokenEl);

	// Disk health
	const diskHealthEl = document.createElement('p');
	diskHealthEl.textContent = `Disk Health: ${data.DiskHealthPcnt ?? 'N/A'}`;
	healthInfoDiv.appendChild(diskHealthEl);

	// Battery health
	const batteryHealthEl = document.createElement('p');
	batteryHealthEl.textContent = `Battery Health: ${data.BatteryHealthPcnt !== null ? `${Math.round(data.BatteryHealthPcnt)}%` : 'N/A'}`;
	healthInfoDiv.appendChild(batteryHealthEl);

	fragment.appendChild(healthInfoDiv);

	clientInfoContainer.appendChild(fragment);
}

document.addEventListener('DOMContentLoaded', async () => {
	const clientData: ClientInfoResponse | null = await fetchClientData();
	if (!clientData) {
		console.error('No client data available');
		return;
	}

	renderClientData(clientData);
});
