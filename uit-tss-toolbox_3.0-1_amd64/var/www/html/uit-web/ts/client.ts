const clientInfoContainer = document.getElementById('client-info-container') as HTMLDivElement | null;
const pageTitle = document.getElementById('page-title') as HTMLHeadingElement | null;
const clientActionsContainer = document.getElementById('client-actions-container') as HTMLDivElement | null;

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
	SecureBootEnabled:         boolean | null
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
	AdminUsers:                string[] | null
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
	BatteryManufactureDate:    Date | null
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
	if (!clientInfoContainer || !pageTitle || !clientActionsContainer) {
		console.error('Client info container, page title, or client actions container element not found');
		return;
	}

	const tag = new URLSearchParams(window.location.search).get('tagnumber');
	document.title = `${tag} - Client Details - UIT Toolbox`;
	pageTitle.textContent = `Client Details - ${tag ?? 'N/A'}`;
	if (!data) {
		console.error('No client data available');
		return;
	}

	// Inventory button link
	const updateInventoryButton = document.createElement('button');
	updateInventoryButton.textContent = 'Update Inventory';
	updateInventoryButton.classList.add('svg-button', 'text-left', 'edit');
	const updateInventoryLink = document.createElement('a');
	const updateInventoryURL = new URL('inventory', window.location.origin);
	updateInventoryURL.searchParams.set('tagnumber', data.Tagnumber?.toString() ?? '');
	updateInventoryURL.searchParams.set('update', 'true');
	updateInventoryLink.href = updateInventoryURL.toString();
	updateInventoryLink.appendChild(updateInventoryButton);
	clientActionsContainer.appendChild(updateInventoryLink);

	// View images link
	const imageViewButton = document.createElement('button');
	imageViewButton.textContent = 'View Client Images';
	if (data.ClientImages && data.ClientImages.length > 0) imageViewButton.textContent += ` (${data.ClientImages.length})`;
	imageViewButton.classList.add('svg-button', 'text-left', 'photo-album');
	const imageViewLink = document.createElement('a');
	const imageViewURL = new URL('client_images', window.location.origin);
	imageViewURL.searchParams.set('tagnumber', data.Tagnumber?.toString() ?? '');
	imageViewLink.href = imageViewURL.toString();
	imageViewLink.appendChild(imageViewButton);
	clientActionsContainer.appendChild(imageViewLink);

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
	ClientUUIDEl.textContent = `Client UUID: `;
	const clientUUIDSpan = document.createElement('span');
	clientUUIDSpan.classList.add('copyable-text');
	clientUUIDSpan.textContent = data.ClientUUID ?? 'N/A';
	ClientUUIDEl.addEventListener('click', () => {
		if (data.ClientUUID) {
			navigator.clipboard.writeText(data.ClientUUID);
			showCopiedTextStyleChange(clientUUIDSpan);
		}
	});
	ClientUUIDEl.appendChild(clientUUIDSpan);
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

	// Computer Name
	const computerNameEl = document.createElement('p');
	computerNameEl.textContent = `Computer Name: ${data.ComputerName ?? 'N/A'}`;
	osInfoDiv.appendChild(computerNameEl);

	if (data.OSInstalled === true && data.DiskRemoved === false) {
		// OS Name
		const osNameEl = document.createElement('p');
		osNameEl.textContent = `OS Name: ${data.OSName ?? 'N/A'}`;
		osInfoDiv.appendChild(osNameEl);

		// OS Version
		const osVersionEl = document.createElement('p');
		osVersionEl.textContent = `OS Version: ${data.OSVersion ?? 'N/A'}`;
		osInfoDiv.appendChild(osVersionEl);

		// BIOS Version
		const biosVersionEl = document.createElement('p');
		if (data.BIOSVersion && data.BIOSReleaseDate) {
			biosVersionEl.textContent = `BIOS Version: ${data.BIOSVersion} (Released: ${new Date(data.BIOSReleaseDate).toLocaleDateString()})`;
		} if (data.BIOSVersion) {
			biosVersionEl.textContent = `BIOS Version: ${data.BIOSVersion}`;
		} else {
			biosVersionEl.textContent = 'BIOS Version: N/A';
		}
		osInfoDiv.appendChild(biosVersionEl);

		// TPM Version and Secure Boot 
		const tpmVersionEl = document.createElement('p');
		tpmVersionEl.textContent = `TPM Version: ${data.TPMVersion ?? 'N/A'}`;
			const secureBootSpan = document.createElement('span');
		if (data.SecureBootEnabled !== null) {
			secureBootSpan.textContent = ` (Secure Boot ${data.SecureBootEnabled ? 'Enabled' : 'Disabled'})`;
			tpmVersionEl.appendChild(secureBootSpan);
		} else {
			secureBootSpan.textContent = ' (Secure Boot Status Unknown)';
			secureBootSpan.style.fontStyle = 'italic';
			tpmVersionEl.appendChild(secureBootSpan);
		}
		osInfoDiv.appendChild(tpmVersionEl);

		// AD Domain / OU
		const ouEl = document.createElement('p');
		const ouSpan = document.createElement('span');
		ouSpan.appendChild(document.createTextNode(`AD OU: ${data.OUName ?? 'N/A'}`));
		ouEl.appendChild(ouSpan);
		if (data.IsIntuneJoined === true) {
			const intuneSpan = document.createElement('span');
			intuneSpan.appendChild(document.createTextNode(' (Intune Joined)'));
			intuneSpan.style.fontStyle = 'italic';
			ouEl.appendChild(intuneSpan);
		} else {
			const notIntuneSpan = document.createElement('span');
			notIntuneSpan.appendChild(document.createTextNode(' (Not Intune Joined)'));
			notIntuneSpan.style.fontStyle = 'italic';
			notIntuneSpan.style.color = 'red';
			ouEl.appendChild(notIntuneSpan);
		}
		osInfoDiv.appendChild(ouEl);

		// Disk Encryption
		const encryptionEl = document.createElement('p');
		encryptionEl.textContent = `Disk Encrypted: ${data.IsDiskEncrypted !== null ? (data.IsDiskEncrypted ? 'Yes' : 'No') : 'N/A'}`;
		osInfoDiv.appendChild(encryptionEl);

		// Admin Users
		const adminUsersEl = document.createElement('p');
		adminUsersEl.textContent = `Admin Users: ${data.AdminUsers && data.AdminUsers.length > 0 ? data.AdminUsers.join(', ') : 'N/A'}`;
		osInfoDiv.appendChild(adminUsersEl);

	} else {
		const osNotInstalledEl = document.createElement('p');
		osNotInstalledEl.appendChild(document.createTextNode('OS Not Installed'));
		if (data.DiskRemoved === true) {
			const osNotInstalledSpanWarn = document.createElement('span');
			osNotInstalledSpanWarn.appendChild(document.createTextNode(' (Disk Removed)'));
			osNotInstalledSpanWarn.style.fontStyle = 'italic';
			osNotInstalledSpanWarn.style.color = 'red';
			osNotInstalledEl.appendChild(osNotInstalledSpanWarn);
		}
		osInfoDiv.appendChild(osNotInstalledEl);
	}

	// Last OS Entry Time
	const lastOsEntryEl = document.createElement('p');
	lastOsEntryEl.textContent = `OS Info Last Updated: ${data.LastOSEntryTime ? new Date(data.LastOSEntryTime).toLocaleString() : 'N/A'}`;
	osInfoDiv.appendChild(lastOsEntryEl);

	fragment.appendChild(osInfoDiv);


	// Client Health
	const healthInfoDiv = document.createElement('div');

	// Is Broken
	const isBrokenEl = document.createElement('p');
	isBrokenEl.textContent = `Is Broken: ${data.IsBroken !== null ? (data.IsBroken ? 'Yes' : 'No') : 'N/A'}`;
	healthInfoDiv.appendChild(isBrokenEl);

	// Disk health
	const diskHealthEl = document.createElement('p');
	const diskHealthSpan = document.createElement('span');
	if (data.DiskHealthPcnt !== null && data.DiskRemoved === false) {
		diskHealthSpan.appendChild(document.createTextNode(`Disk Health: ${data.DiskHealthPcnt.toFixed(2)}%`));
	} else if (data.DiskRemoved === true) {
		diskHealthSpan.appendChild(document.createTextNode(`Disk Health: N/A (Disk Removed)`));
	} else {
		diskHealthSpan.appendChild(document.createTextNode('Disk Health: N/A (Missing Data)'));
	}
	diskHealthEl.appendChild(diskHealthSpan);
	healthInfoDiv.appendChild(diskHealthEl);

	// Battery health
	const batteryHealthEl = document.createElement('p');
	const batteryHealthSpan = document.createElement('span');
	if (data.BatteryHealthPcnt !== null) {
		batteryHealthSpan.appendChild(document.createTextNode(`Battery Health: ${data.BatteryHealthPcnt.toFixed(2)}%`));
	} else {
		batteryHealthSpan.appendChild(document.createTextNode('Battery Health: N/A'));
	}
	batteryHealthEl.appendChild(batteryHealthSpan);
	healthInfoDiv.appendChild(batteryHealthEl);

	// Last Hardware Check
	const lastHardwareCheckEl = document.createElement('p');
	lastHardwareCheckEl.textContent = `Last Hardware Check: ${data.LastHardwareCheck ? new Date(data.LastHardwareCheck).toLocaleString() : 'N/A'}`;
	healthInfoDiv.appendChild(lastHardwareCheckEl);


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
