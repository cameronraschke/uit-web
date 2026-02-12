let updatingInventory = false;

type InventoryForm = {
	time: string | null;
	tagnumber: number | null;
	system_serial: string | null;
	location: string | null;
	building: string | null;
	room: string | null;
	system_manufacturer: string | null;
	system_model: string | null;
	device_type: string | null;
	department_name: string | null;
	ad_domain: string | null;
	property_custodian: string | null;
	acquired_date: Date | null;
	retired_date: Date | null;
	is_broken: boolean | null;
	disk_removed: boolean | null;
	last_hardware_check: Date | null;
	status: string | null;
	checkout_bool: boolean | null;
	checkout_date: Date | null;
	return_date: Date | null;
	note: string | null;
};

type Department = {
	department_name: string;
	department_name_formatted: string;
	department_sort_order: number;
	organization_name: string;
	organization_name_formatted: string;
	organization_sort_order: number;
};

type DepartmentsCache = {
	timestamp: number;
	departments: Department[];
};

type ClientLookupResult = {
	tagnumber: number | null;
	system_serial: string | null;
};

type AllLocations = {
	timestamp: Date | null;
	location: string | null;
	location_formatted: string | null;
	location_count: number | null;
};

type AllLocationsCache = {
	timestamp: number;
	locations: AllLocations[];
};

// Inventory lookup form elements
const updateFormContainer = document.getElementById('inventory-form-container') as HTMLElement;
const clientLookupWarningMessage = document.getElementById('existing-inventory-message') as HTMLElement;
const clientLookupForm = document.getElementById('inventory-lookup-form') as HTMLFormElement;
const clientLookupTagInput = document.getElementById('inventory-tag-lookup') as HTMLInputElement;
const clientLookupSerial = document.getElementById('inventory-serial-lookup') as HTMLInputElement;
const clientLookupSubmitButton = document.getElementById('inventory-lookup-submit-button') as HTMLButtonElement;
const clientLookupReset = document.getElementById('inventory-lookup-reset-button') as HTMLButtonElement;
const clientMoreDetails = document.getElementById('inventory-lookup-more-details') as HTMLButtonElement;
const clientViewPhotos = document.getElementById('inventory-lookup-photo-album') as HTMLButtonElement;
const clientAddPhotos = document.getElementById('inventory-lookup-add-photos') as HTMLButtonElement;
const allTagsDatalist = document.getElementById('inventory-tag-suggestions') as HTMLDataListElement;
const advSearchDepartment = document.getElementById('adv-search-department') as HTMLSelectElement;
const advSearchDomain = document.getElementById('adv-search-ad-domain') as HTMLSelectElement;
const advSearchStatus = document.getElementById('adv-search-status') as HTMLSelectElement;
const advSearchDeviceType = document.getElementById('adv-search-device-type') as HTMLSelectElement;
const csvDownloadButton = document.getElementById('adv-search-download-csv') as HTMLButtonElement;
const printCheckoutLink = document.getElementById('print-checkout-link') as HTMLElement;
const printCheckoutContainer = document.getElementById('print-checkout-container') as HTMLElement;

// Inventory update form elements
const updateForm = document.getElementById('inventory-update-form') as HTMLFormElement;
const lastUpdateTime = document.getElementById('last-update-time-message') as HTMLElement;
const locationEl = document.getElementById('location') as HTMLInputElement;
const buildingUpdate = updateForm.querySelector("#building") as HTMLInputElement;
const roomUpdate = updateForm.querySelector("#room") as HTMLInputElement;
const manufacturerUpdate = updateForm.querySelector("#system_manufacturer") as HTMLInputElement;
const modelUpdate = updateForm.querySelector("#system_model") as HTMLInputElement;
const deviceTypeUpdate = updateForm.querySelector("#device_type") as HTMLSelectElement;
const departmentEl = document.getElementById('department_name') as HTMLSelectElement;
const adDomainUpdate = updateForm.querySelector("#ad_domain") as HTMLSelectElement;
const propertyCustodianUpdate = updateForm.querySelector("#property_custodian") as HTMLInputElement;
const acquiredDateUpdate = updateForm.querySelector("#acquired_date") as HTMLInputElement;
const retiredDateUpdate = updateForm.querySelector("#retired_date") as HTMLInputElement;
const isBrokenUpdate = updateForm.querySelector("#is_broken") as HTMLSelectElement;
const diskRemovedUpdate = updateForm.querySelector("#disk_removed") as HTMLSelectElement;
const lastHardwareCheckUpdate = updateForm.querySelector("#last_hardware_check") as HTMLInputElement;
const clientStatusUpdate = updateForm.querySelector("#status") as HTMLSelectElement;
const checkoutBoolUpdate = updateForm.querySelector("#checkout_bool") as HTMLSelectElement;
const checkoutDateUpdate = updateForm.querySelector("#checkout_date") as HTMLInputElement;
const returnDateUpdate = updateForm.querySelector("#return_date") as HTMLInputElement;
const noteUpdate = updateForm.querySelector("#note") as HTMLInputElement;
const fileInputUpdate = updateForm.querySelector("#inventory-file-input") as HTMLInputElement;
const submitUpdate = document.getElementById('inventory-update-submit-button') as HTMLButtonElement;
const cancelUpdate = document.getElementById('inventory-update-cancel-button') as HTMLButtonElement;

const allowedFileNameRegex = /^[a-zA-Z0-9.\-_ ()]+\.[a-zA-Z]+$/; // file name + extension
const allowedFileExtensions = [".jpg", ".jpeg", ".jfif", ".png"];

const statusesThatIndicateBroken = ["needs-repair"];
const statusesThatIndicateCheckout = ["checked-out", "reserved-for-checkout"];

const allInventoryUpdateFields = [
	clientLookupTagInput,
	clientLookupSerial,
	locationEl,
	buildingUpdate,
	roomUpdate,
	manufacturerUpdate,
	modelUpdate,
	deviceTypeUpdate,
	departmentEl,
	adDomainUpdate,
	propertyCustodianUpdate,
	acquiredDateUpdate,
	retiredDateUpdate,
	isBrokenUpdate,
	diskRemovedUpdate,
	lastHardwareCheckUpdate,
	clientStatusUpdate,
	checkoutBoolUpdate,
	checkoutDateUpdate,
	returnDateUpdate,
	noteUpdate,
	fileInputUpdate,
];

const requiredInventoryUpdateFields = [
	clientLookupTagInput,
	clientLookupSerial,
	locationEl,
	departmentEl,
	adDomainUpdate,
	clientStatusUpdate
];

const buttonsVisibleWhenUpdating = [
	clientMoreDetails,
	clientViewPhotos,
	clientAddPhotos,
];

async function fetchAllLocations(purgeCache: boolean = false): Promise<AllLocations[] | []> {
	const cached = sessionStorage.getItem("uit_all_locations");

	try {
		if (cached && !purgeCache) {
			const cacheEntry: AllLocationsCache = JSON.parse(cached);
			if (Date.now() - cacheEntry.timestamp < 300000 && Array.isArray(cacheEntry.locations)) {
				console.log("Loaded all locations from cache");
				return cacheEntry.locations;
			}
		}
		const data: AllLocations[] = await fetchData('/api/locations', false);
		if (!data || !Array.isArray(data)) {
			throw new Error("No data returned from /api/locations");
		}
		sessionStorage.setItem("uit_all_locations", JSON.stringify({ timestamp: Date.now(), locations: data }));
		return data;
	} catch (error) {
		const errorMessage = error instanceof Error ? error.message : String(error);
		console.error("Error fetching all locations:", errorMessage);
		return [];
	}
}

function getSortedLocations(inputElement: HTMLInputElement, data: Array<AllLocations>): Array<{ location: string, location_formatted: string | null }> {
	if (!inputElement || !data || data.length === 0) {
		return [];
	}

	const charsToTrim = new RegExp(['"', "'", '`', ' '].join(''), 'g');
	const inputValue = inputElement.value;
	const inputValueStripped = inputValue.trim().toLowerCase().replaceAll(charsToTrim, '');

	return data
		.filter((entry) => {
			if (typeof entry.location !== 'string') {
				console.warn('Data entry location is not a string:', entry);
				return false;
			}
			if (inputValue === entry.location) {
				return false; // Return early on match
			}
			const strippedLocation = entry.location.trim().toLowerCase().replaceAll(charsToTrim, '');
			return strippedLocation.includes(inputValueStripped);
		})
		.sort((a, b) => {
			const timestampA = a.timestamp ? new Date(a.timestamp).getTime() : 0;
			const timestampB = b.timestamp ? new Date(b.timestamp).getTime() : 0;
			return timestampB - timestampA;
		})
		.map(entry => ({
			location: entry.location!,
			location_formatted: entry.location_formatted
		}))
		.slice(0, 10);
}

async function lookupTagOrSerial(tagnumber: number | null, serial: string | null): Promise<ClientLookupResult | null> {
  const query = new URLSearchParams();
  if (tagnumber) {
    query.append("tagnumber", tagnumber.toString());
  } else if (serial) {
    query.append("system_serial", serial);
  } else {
    console.log("No tag or serial provided");
    return null;
  }
  try {
    const data = await fetchData(`/api/lookup?${query.toString()}`);
    if (!data) {
      console.log("No data returned from /api/lookup");
			return null;
    }
		const jsonResponse: ClientLookupResult = data as ClientLookupResult;
		if (!jsonResponse || (jsonResponse.tagnumber === null && !jsonResponse.system_serial)) {
			console.log("No data found for provided tag or serial");
			return null;
		}
    return jsonResponse;
  } catch(error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.log("Error getting tag/serial: " + errorMessage);
		return null;
  }
}

async function submitInventoryLookup() {
	updateURLFromFilters();
	const searchParams: URLSearchParams = new URLSearchParams(window.location.search);
	const lookupTag: number | null = clientLookupTagInput.value ? Number(clientLookupTagInput.value) : (searchParams.get('tagnumber') ? Number(searchParams.get('tagnumber')) : null);
  const lookupSerial: string | null = clientLookupSerial.value || searchParams.get('system_serial') || null;

	clientLookupTagInput.dataset.initialValue = clientLookupTagInput.value;
	clientLookupSerial.dataset.initialValue = clientLookupSerial.value;

  if (!lookupTag && !lookupSerial) {
    clientLookupWarningMessage.style.display = "block";
    clientLookupWarningMessage.textContent = "Please provide a tag number or serial number to look up.";
    return;
  }
  if (lookupTag && isNaN(Number(lookupTag))) {
    clientLookupWarningMessage.style.display = "block";
    clientLookupWarningMessage.textContent = "Tag number must be numeric.";
    return;
  }
  if (lookupSerial && (lookupSerial.length < 4 || lookupSerial.length > 20)) {
    clientLookupWarningMessage.style.display = "block";
    clientLookupWarningMessage.textContent = "Serial number must be between 4 and 20 characters long.";
    return;
  }
  if (lookupTag && lookupTag.toString().length != 6) {
    clientLookupWarningMessage.style.display = "block";
    clientLookupWarningMessage.textContent = "Tag number must be exactly 6 digits long.";
    return;
  }

	clientLookupReset.style.display = "inline-block";
	for (const btn of buttonsVisibleWhenUpdating) {
		btn.style.display = "inline-block";
		btn.disabled = false;
	}

	try {
		const lookupResult: ClientLookupResult | null = await lookupTagOrSerial(lookupTag, lookupSerial);

		if (lookupResult) {
			if (lookupResult.tagnumber && !isNaN(Number(lookupResult.tagnumber))) {
				searchParams.set("tagnumber", lookupResult.tagnumber ? lookupResult.tagnumber.toString() : '');
				clientLookupTagInput.value = Number(lookupResult.tagnumber).toString();
				clientLookupTagInput.dataset.initialValue = lookupResult.tagnumber ? lookupResult.tagnumber.toString() : "";
				clientLookupTagInput.readOnly = true;
			}
			if (lookupResult.system_serial && lookupResult.system_serial && lookupResult.system_serial.trim().length > 0) {
				searchParams.set("system_serial", lookupResult.system_serial ? lookupResult.system_serial.trim() : '');
				clientLookupSerial.value = lookupResult.system_serial.trim();
				clientLookupSerial.dataset.initialValue = lookupResult.system_serial ? lookupResult.system_serial : "";
				clientLookupSerial.readOnly = true;
			}

			clientLookupSubmitButton.disabled = true;
			clientLookupSubmitButton.style.cursor = "not-allowed";
			clientLookupSubmitButton.style.border = "1px solid gray";
			clientLookupSubmitButton.style.backgroundColor = "lightgray";
			clientLookupSubmitButton.style.display = "none";

			for (const btn of buttonsVisibleWhenUpdating) {
				btn.disabled = false;
				btn.style.display = "inline-block";
				btn.style.cursor = "pointer";
			}

			if (lookupResult.tagnumber || lookupResult.system_serial) {
				await populateLocationForm(lookupResult.tagnumber ? lookupResult.tagnumber : undefined, lookupResult.system_serial ? lookupResult.system_serial : undefined);
			}
		} else {
			clientLookupWarningMessage.style.display = "block";
			clientLookupWarningMessage.textContent = "No inventory entry was found for the provided tag number or serial number. A new entry can be created.";

			for (const btn of buttonsVisibleWhenUpdating) {
				btn.disabled = true;
				btn.style.cursor = "not-allowed";
			}
			const tagNum = clientLookupTagInput.value ? Number(clientLookupTagInput.value) : '';
			const serialNum = clientLookupSerial.value ? clientLookupSerial.value : '';
			searchParams.set("tagnumber", tagNum.toString());
			searchParams.set("system_serial", serialNum);
			await populateLocationForm(tagNum ? tagNum : undefined, serialNum ? serialNum : undefined);
		}
	} catch (error) {
		const errorMessage = error instanceof Error ? error.message : String(error);
		console.error("Error during inventory lookup: " + errorMessage);
		clientLookupWarningMessage.style.display = "block";
		clientLookupWarningMessage.textContent = "Error looking up inventory entry. Check console.";
	} finally {
		// Set 'update' parameter in URL
		searchParams.set('update', 'true');
		history.replaceState(null, '', window.location.pathname + '?' + searchParams.toString());
	}
}

async function updateCheckoutStatus() {
	if (statusesThatIndicateCheckout.includes(clientStatusUpdate.value)) {
		printCheckoutContainer.style.display = 'inline-block';
		printCheckoutLink.setAttribute('href', `/checkout-form?tagnumber=${encodeURIComponent(clientLookupTagInput.value)}`);
		printCheckoutLink.setAttribute('target', '_blank');
		printCheckoutLink.textContent = 'Print Checkout Form';
	} else {
		printCheckoutContainer.style.display = 'none';
		printCheckoutLink.removeAttribute('href');
		printCheckoutLink.removeAttribute('target');
		printCheckoutLink.textContent = '';
	}
}

function resetInventoryLookupAndUpdateForm() {
	clientLookupForm.reset();
	updateForm.reset();
	setURLParameter('update', null);
	setURLParameter('tagnumber', 	null);
	setURLParameter('system_serial', null);
	for (const el of allInventoryUpdateFields) {
		if (el instanceof HTMLInputElement) {
			resetInputElement(el, "", false, undefined);
		}
		if (el instanceof HTMLSelectElement) {
			resetSelectElement(el, "", false, undefined);
		}
	}
	clientLookupTagInput.placeholder = "Enter Tag Number";
	clientLookupSerial.placeholder = "Enter System Serial";

	clientLookupSubmitButton.disabled = false;
	clientLookupSubmitButton.style.cursor = "pointer";
	clientLookupSubmitButton.style.border = "1px solid black";
	clientLookupSubmitButton.style.backgroundColor = "";
	clientLookupSubmitButton.style.display = "inline-block";

	clientLookupReset.style.display = "none";

	for (const btn of buttonsVisibleWhenUpdating) {
		btn.style.display = "none";
		btn.disabled = false;
		btn.style.cursor = "pointer";
	}
	updateForm.style.display = "none";
	clientLookupWarningMessage.style.display = "none";
	clientLookupWarningMessage.textContent = "";
	lastUpdateTime.textContent = "";
	lastUpdateTime.style.display = "none";
	clientLookupTagInput.focus();
}

function resetInventorySearchQuery() {
	const url = new URL(window.location.pathname, window.location.origin);
	url.searchParams.delete('tagnumber');
	url.searchParams.delete('system_serial');
	url.searchParams.delete('update');
	history.replaceState(null, '', url.toString());
}

function renderTagOptions(tags: number[]): void {
  if (!allTagsDatalist) {
    console.warn("No tag datalist found");
    return;
  }
  
  allTagsDatalist.innerHTML = '';
  let maxTags = 20;
	if (tags.length < maxTags) {
		maxTags = tags.length;
	}
  (tags || []).slice(0, maxTags).forEach(tag => {
    const option = document.createElement('option');
    option.value = tag.toString();
    allTagsDatalist.appendChild(option);
  });
}

async function getLocationFormData(tag?: number, serial?: string): Promise<InventoryForm | null> {
	const tagNum = tag ? tag : clientLookupTagInput.value ? Number(clientLookupTagInput.value) : null;
	const serialNum = serial ? serial : clientLookupSerial.value ? String(clientLookupSerial.value) : null;
	const url = new URL('/api/client/location_form_data', window.location.origin);
	url.searchParams.set('tagnumber', tagNum !== null ? tagNum.toString() : '');
	url.searchParams.set('system_serial', serialNum !== null ? serialNum : '');

  try {
    const response: InventoryForm = await fetchData(url.toString(), false);
    if (!response) {
      throw new Error("Cannot parse json from /api/client/location_form_data");
    }
    return response;
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.log("Error fetching location form data: " + errorMessage);
    return null;
  }
}

function showInventoryUpdateChanges(): void {
	const inputs = updateForm.querySelectorAll("input, select, textarea");

	inputs.forEach((el: HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement) => {
		el.dataset.initialValue = el.value;

		const handleInputUpdate = () => {
			if (updatingInventory) return;

			// Check if value matches the initial value, unless it's required + blank
			if (el.dataset.initialValue === el.value && !(el.required && el.value.trim() === "")) {
				for (const cssClass of inputCSSClasses) {
					el.classList.remove(cssClass);
				}
				return;
			}

			// If different, update classes
			for (const cssClass of inputCSSClasses) {
				el.classList.remove(cssClass);
			}

			// If required and blank
			if (el.required && el.value.trim() === "") {
				el.classList.add("empty-required-input");
			} else {
				el.classList.add("changed-input");
			}
		};

		el.oninput = handleInputUpdate;
		el.onchange = handleInputUpdate;
	});
}

async function populateLocationForm(tag?: number, serial?: string): Promise<void> {
	// reset/zero/clear out all fields before processing new data
	resetInputElement(clientLookupTagInput, "Enter Tag Number", false, undefined);
	clientLookupTagInput.value = clientLookupTagInput.dataset.initialValue || "";
	if (clientLookupTagInput.value) {
		clientLookupTagInput.readOnly = true;
		clientLookupTagInput.value = clientLookupTagInput.value.toString().trim();
		clientLookupTagInput.classList.add("readonly-input");
	} else {
		clientLookupTagInput.classList.add("empty-required-input");
	}

	resetInputElement(clientLookupSerial, "Enter System Serial", false, undefined);
	clientLookupSerial.value = clientLookupSerial.dataset.initialValue || "";
	if (clientLookupSerial.value) {
		clientLookupSerial.readOnly = true;
		clientLookupSerial.value = clientLookupSerial.value.trim();
		clientLookupSerial.classList.add("readonly-input");
	} else {
		clientLookupSerial.classList.add("empty-required-input");
	}

	resetInputElement(locationEl, "Enter Location", false, "empty-required-input");

	resetInputElement(buildingUpdate, "Building", false, "empty-input");

	resetInputElement(roomUpdate, "Room", false, "empty-input");

	resetInputElement(manufacturerUpdate, "System Manufacturer", false, "empty-input");

	resetInputElement(modelUpdate, "System Model", false, "empty-input");

	resetSelectElement(deviceTypeUpdate, "Device Type", false, "empty-input");
	try {
		await populateDeviceTypeSelect(deviceTypeUpdate);
		deviceTypeUpdate.classList.add("empty-input");
	} catch(e) {
		const errorMessage = e instanceof Error ? e.message : String(e);
		console.error(`Could not fetch all device types: ${errorMessage}`)
	}

	resetSelectElement(departmentEl, "Select Department", false, "empty-required-input");
	try { 
		await populateDepartmentSelect(departmentEl)
		departmentEl.classList.add("empty-required-input");
	} catch(e) {
		const errorMessage = e instanceof Error ? e.message : String(e);
		console.error(`Could not fetch all departments: ${errorMessage}`)
	}

	resetSelectElement(adDomainUpdate, "Select Domain", false, "empty-required-input");
	try { 
		await populateDomainSelect(adDomainUpdate);
		adDomainUpdate.classList.add("empty-required-input");
	} catch(e) {
		const errorMessage = e instanceof Error ? e.message : String(e);
		console.error(`Could not fetch all domains: ${errorMessage}`)
	}

	resetInputElement(propertyCustodianUpdate, "Property Custodian", false, "empty-input");

	resetInputElement(acquiredDateUpdate, "Acquired Date", false, "empty-input");

	resetInputElement(retiredDateUpdate, "Retired Date", false, "empty-input");

	resetSelectElement(isBrokenUpdate, "Is Broken?", false, "empty-input");
		if (isBrokenUpdate) {
		const op1 = document.createElement("option");
		op1.value = "true";
		op1.textContent = "Is broken"

		const op2 = document.createElement("option");
		op2.value = "false";
		op2.textContent = "Is functional";

		const op3 = document.createElement("option");
		op3.value = "unknown";
		op3.textContent = "Unknown";

		isBrokenUpdate.append(op1);
		isBrokenUpdate.append(op2);
		isBrokenUpdate.append(op3);
	}

	resetSelectElement(diskRemovedUpdate, "Disk Removed?", false, "empty-input");
	if (diskRemovedUpdate) {
		const op1 = document.createElement("option");
		op1.value = "true";
		op1.textContent = "Yes, disk removed"

		const op2 = document.createElement("option");
		op2.value = "false";
		op2.textContent = "No, disk present";

		const op3 = document.createElement("option");
		op3.value = "unknown";
		op3.textContent = "Unknown";

		diskRemovedUpdate.append(op1);
		diskRemovedUpdate.append(op2);
		diskRemovedUpdate.append(op3);
	}

	resetInputElement(lastHardwareCheckUpdate, "Last Hardware Check", false, "empty-input");

	resetSelectElement(clientStatusUpdate, "Select Client Status", false, "empty-required-input");
	try { 
		await populateStatusSelect(clientStatusUpdate);
		clientStatusUpdate.classList.add("empty-required-input");
	} catch(e) {
		const errorMessage = e instanceof Error ? e.message : String(e);
		console.error(`Could not fetch all statuses: ${errorMessage}`)
	}

	// Checkout date omitted - included during form submit
	
	resetInputElement(checkoutDateUpdate, "Checkout Date", false, "empty-input");

	resetInputElement(returnDateUpdate, "Return Date", false, "empty-input");

	resetInputElement(fileInputUpdate, "", false, undefined);

	resetInputElement(noteUpdate, "Enter Note", false, "empty-input");

	for (const el of requiredInventoryUpdateFields) {
		el.required = true;
	}

	lastUpdateTime.textContent = "";
	lastUpdateTime.style.display = "none";
	try {
		const locationFormData = await getLocationFormData(tag, serial);
		if (locationFormData) {
			if (locationFormData.time) {
				const lastUpdate = new Date(locationFormData.time);
				const timeFormattingOptions : Intl.DateTimeFormatOptions = {
					hour: "2-digit",
					minute: "2-digit",
					weekday: "short",
					year: "numeric",
					month: "long",
					day: "numeric",
				};
				if (isNaN(lastUpdate.getTime())) {
					lastUpdateTime.textContent = 'Unknown timestamp of last entry';
				} else {
					lastUpdateTime.textContent = `Most recent entry: ${lastUpdate.toLocaleString(undefined, timeFormattingOptions)}` || '';
				}
				lastUpdateTime.style.display = "block";
			}
		
			if (locationFormData.tagnumber) {
				clientLookupTagInput.value = locationFormData.tagnumber.toString();
				clientLookupTagInput.classList.remove("empty-required-input");
				clientLookupTagInput.classList.add("readonly-input");
				clientLookupTagInput.readOnly = true;
			} else {
				clientLookupTagInput.classList.add("empty-required-input");
			}

			if (locationFormData.system_serial) {
				clientLookupSerial.value = locationFormData.system_serial.trim();
				clientLookupSerial.classList.remove("empty-required-input");
				clientLookupSerial.classList.add("readonly-input");
				clientLookupSerial.readOnly = true;
			} else {
				clientLookupSerial.classList.add("empty-required-input");
			}

			if (locationFormData.location ) {
				locationEl.value = locationFormData.location.trim();
				locationEl.classList.remove("empty-required-input");
			} else {
				locationEl.classList.add("empty-required-input");
			}

			if (locationFormData.building) {
				buildingUpdate.value = locationFormData.building.trim();
				buildingUpdate.classList.remove("empty-input");
			} else {
				buildingUpdate.classList.add("empty-input");
			}

			if (locationFormData.room) {
				roomUpdate.value = locationFormData.room.trim();
				roomUpdate.classList.remove("empty-input");
			} else {
				roomUpdate.classList.add("empty-input");
			}

			if (locationFormData.system_manufacturer) {
				manufacturerUpdate.readOnly = true;
				manufacturerUpdate.value = locationFormData.system_manufacturer.trim();
				manufacturerUpdate.classList.remove("empty-input");
				manufacturerUpdate.classList.add("readonly-input");
			} else {
				manufacturerUpdate.classList.add("empty-input");
			}

			if (locationFormData.system_model) {
				modelUpdate.readOnly = true;
				modelUpdate.value = locationFormData.system_model.trim();
				modelUpdate.classList.remove("empty-input");
				modelUpdate.classList.add("readonly-input");
			} else {
				modelUpdate.classList.add("empty-input");
			}

			if (locationFormData.device_type) {
				deviceTypeUpdate.value = locationFormData.device_type.trim();
				deviceTypeUpdate.classList.remove("empty-input");
			} else {
				deviceTypeUpdate.classList.add("empty-input");
			}

			if (locationFormData.department_name) {
				departmentEl.value = locationFormData.department_name.trim();
				departmentEl.classList.remove("empty-required-input");
			} else {
				departmentEl.classList.add("empty-required-input");
			}

			if (locationFormData.ad_domain) {
				adDomainUpdate.value = locationFormData.ad_domain.trim();
				adDomainUpdate.classList.remove("empty-required-input");
			} else {
				adDomainUpdate.classList.add("empty-required-input");
			}

			if (locationFormData.property_custodian) {
				propertyCustodianUpdate.value = locationFormData.property_custodian.trim();
				propertyCustodianUpdate.classList.remove("empty-input");
			} else {
				propertyCustodianUpdate.classList.add("empty-input");
			}

			if (locationFormData.acquired_date) {
				const acquiredDateValue = locationFormData.acquired_date ? new Date(locationFormData.acquired_date) : null;
				if (acquiredDateValue && !isNaN(acquiredDateValue.getTime())) {
					// Format as YYYY-MM-DD for input[type="date"]
					const year = acquiredDateValue.getFullYear();
					const month = String(acquiredDateValue.getMonth() + 1).padStart(2, '0');
					const day = String(acquiredDateValue.getDate()).padStart(2, '0');
					const acquiredDateFormatted = `${year}-${month}-${day}`;
					acquiredDateUpdate.value = acquiredDateFormatted;
					acquiredDateUpdate.classList.remove("empty-input");
				}
			} else {
				acquiredDateUpdate.classList.add("empty-input");
			}
			
			if (locationFormData.is_broken === true) {
				isBrokenUpdate.value = "true";
				isBrokenUpdate.classList.remove("empty-input");
			} else if (locationFormData.is_broken === false) {
				isBrokenUpdate.value = "false";
				isBrokenUpdate.classList.remove("empty-input");
			} else {
				isBrokenUpdate.classList.add("empty-input");
			}

			if (locationFormData.disk_removed === true) {
				diskRemovedUpdate.value = "true";
				diskRemovedUpdate.classList.remove("empty-input");
			} else if (locationFormData.disk_removed === false) {
				diskRemovedUpdate.value = "false";
				diskRemovedUpdate.classList.remove("empty-input");
			} else {
				diskRemovedUpdate.classList.add("empty-input");
			}

			if (locationFormData.status) {
				clientStatusUpdate.value = locationFormData.status.trim();
				clientStatusUpdate.classList.remove("empty-required-input");
			} else {
				clientStatusUpdate.classList.add("empty-required-input");
			}

			if (locationFormData.note) {
				noteUpdate.value = locationFormData.note.trim();
				noteUpdate.classList.remove("empty-input");
			} else {
				noteUpdate.classList.add("empty-input");
			}
		} else {
			console.warn("No location form data returned from server");
			clientLookupTagInput.dataset.initialValue = clientLookupTagInput.value;
			clientLookupSerial.dataset.initialValue = clientLookupSerial.value;
		}
		await updateCheckoutStatus();
	} catch (error) {
		const errorMessage = error instanceof Error ? error.message : String(error);
		console.error("Error populating location form:", errorMessage);
	} finally {
		showInventoryUpdateChanges();
		updateForm.style.display = "block";
		if (!clientLookupTagInput.value) clientLookupTagInput.focus();
		else if (!clientLookupSerial.value) clientLookupSerial.focus();
		else locationEl.focus();
	}
}

async function fetchDepartments(purgeCache: boolean = false): Promise<Array<Department> | []> {
	const cached = sessionStorage.getItem("uit_departments");

	try {
		if (cached && !purgeCache) {
			const cacheEntry: DepartmentsCache = JSON.parse(cached);
			if (Date.now() - cacheEntry.timestamp < 300000 && Array.isArray(cacheEntry.departments)) {
				console.log("Loaded departments from cache");
				return cacheEntry.departments;
			}
		}
		const data: Array<Department> = await fetchData('/api/departments');
		if (!data || !Array.isArray(data) || data.length === 0) {
			throw new Error('No data returned from /api/departments');
		}
		const cacheEntry: DepartmentsCache = {
			timestamp: Date.now(),
			departments: data
		};
		sessionStorage.setItem("uit_departments", JSON.stringify(cacheEntry));
		console.log("Cached departments data");
		return data;
	} catch (error) {
		console.error('Error fetching departments:', error);
		return [];
	}
}

async function fetchAllDeviceTypes(purgeCache: boolean = false): Promise<DeviceType[] | []> {
	const cached = sessionStorage.getItem("uit_device_types_cache");

	try {
		if (cached && !purgeCache) {
			const cacheEntry: DeviceTypeCache = JSON.parse(cached);
			if (Date.now() - cacheEntry.timestamp < 300000 && Array.isArray(cacheEntry.deviceTypes)) {
				console.log("Loaded device types from cache");
				return cacheEntry.deviceTypes;
			}
		}
		const allDeviceTypes: DeviceType[] = await fetchData(`/api/all_device_types`, false);
		if (!Array.isArray(allDeviceTypes)) {
			throw new Error("returned data is not an array.");
		}
		const cacheEntry: DeviceTypeCache = {
			deviceTypes: allDeviceTypes,
			timestamp: Date.now()
		};
		sessionStorage.setItem("uit_device_types_cache", JSON.stringify(cacheEntry));
		return allDeviceTypes;
	} catch(e) {
		console.error(`Error fetching all device types: ${e}`);
		return [];
	}
}

async function populateDeviceTypeSelect(selectEl: HTMLSelectElement, purgeCache: boolean = false): Promise<void> {
	try {
		const deviceTypes = await fetchAllDeviceTypes(purgeCache);
		if (!deviceTypes || !Array.isArray(deviceTypes)) {
			throw new Error("Invalid device types data");
		}
		const defaultOption = document.createElement("option");
		defaultOption.value = "";
		defaultOption.textContent = "Device Type";
		defaultOption.selected = true;
		defaultOption.disabled = true;
		selectEl.appendChild(defaultOption);
		selectEl.addEventListener('click', () => {
			defaultOption.disabled = true;
		});
		for (const deviceType of deviceTypes) {
			if (!deviceType.device_type || !deviceType.device_type_formatted) {
				console.warn("Skipping invalid device type entry:", deviceType);
				continue;
			}
			const option = document.createElement("option");
			option.value = deviceType.device_type;
			option.textContent = deviceType.device_type_formatted;
			selectEl.appendChild(option);
		}
	} catch (error) {
		console.error("Error populating device type select:", error);
	}
}

async function initializeInventoryPage() {
	const urlParams = new URLSearchParams(window.location.search);

	for (const param of advSearchParams) {
		if (!param.inputElement || !param.paramString) continue;
		initializeAdvSearchListeners(param.inputElement, param.resetElement);
		param.inputElement.dataset.initialValue = urlParams.get(param.paramString) || "";
	}
	filterModel.disabled = !filterManufacturer.value;

	try {
		await Promise.all([
			populateLocationSelect(advSearchLocation, true),
			populateDepartmentSelect(advSearchDepartment, true),
			populateManufacturerSelect(filterManufacturer, filterManufacturerReset, true).then(() => populateModelSelect(filterModel, filterModelReset, true)),
			populateDomainSelect(advSearchDomain, true),
			populateStatusSelect(advSearchStatus),
			populateDeviceTypeSelect(advSearchDeviceType, true)
		]);

		for (const param of advSearchParams) {
			if (!param.inputElement || !param.paramString) continue;
			param.inputElement.value = param.inputElement.dataset.initialValue || "";
			handleAdvSearchInputChange(param.inputElement, param.resetElement);
		}

		// Check URL parameters for auto lookup
		const updateParam: string | null = urlParams.get('update');
		const tagnumberParam: string | null = urlParams.get('tagnumber');
		const systemSerialParam: string | null = urlParams.get('system_serial');
		if (updateParam === 'true') {
			clientLookupTagInput.value = tagnumberParam ? tagnumberParam : '';
			clientLookupSerial.value = systemSerialParam ? systemSerialParam : '';
			await submitInventoryLookup();
			formAnchor.scrollIntoView({ behavior: 'auto', block: 'start' });
		}
		await renderInventoryTable(); // after all URL param handling is complete - lookup, update form, and advanced filters
	} catch (e) {
		const errorMessage = e instanceof Error ? e.message : String(e);
		console.error("Error initializing inventory page:", errorMessage);
	}
}

cancelUpdate.addEventListener("click", (event) => {
	event.preventDefault();
	resetInventoryLookupAndUpdateForm();
	updateURLFromFilters();
});

clientLookupTagInput.addEventListener("keyup", (event: KeyboardEvent) => {
  const searchTerm = ((event.target as HTMLInputElement).value || '').trim().toLowerCase();
  const allTags = Array.isArray(window.allTags) ? window.allTags : [];
  const filteredTags = searchTerm
    ? allTags.filter(tag => tag.toString().trim().includes(searchTerm))
    : allTags;
  if (filteredTags.includes(Number(searchTerm))) {
    allTagsDatalist.innerHTML = '';
  } else {
    renderTagOptions(filteredTags);
  }
});

updateForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  submitUpdate.disabled = true;
  if (updatingInventory) return;
  updatingInventory = true;

	updateURLFromFilters();

  try {
		const formObj = {} as InventoryForm;
		if (!clientLookupTagInput && !clientLookupSerial) {
      throw new Error("No tag or serial input fields found in DOM");
    }
    formObj.tagnumber = getInputNumberValue(clientLookupTagInput);
    formObj.system_serial = getInputStringValue(clientLookupSerial);
		formObj.location = getInputStringValue(locationEl);
		formObj.building = getInputStringValue(buildingUpdate);
		formObj.room = getInputStringValue(roomUpdate);
    formObj.system_manufacturer = getInputStringValue(manufacturerUpdate);
    formObj.system_model = getInputStringValue(modelUpdate);
    formObj.department_name = getInputStringValue(departmentEl);
    formObj.ad_domain = getInputStringValue(adDomainUpdate);
		formObj.property_custodian = getInputStringValue(propertyCustodianUpdate);
		formObj.acquired_date = getInputDateValue(acquiredDateUpdate, true);
		formObj.retired_date = getInputDateValue(retiredDateUpdate, true);
    formObj.is_broken = getInputBooleanValue(isBrokenUpdate);
		formObj.disk_removed = getInputBooleanValue(diskRemovedUpdate);
		formObj.last_hardware_check = getInputTimeValue(lastHardwareCheckUpdate);
    formObj.status = getInputStringValue(clientStatusUpdate);
		formObj.checkout_bool = formObj.status && statusesThatIndicateCheckout.includes(formObj.status) ? true : false;
		formObj.checkout_date = getInputDateValue(checkoutDateUpdate, true);
		formObj.return_date = getInputDateValue(returnDateUpdate, true);
    formObj.note = getInputStringValue(noteUpdate);
    // const jsonBase64 = jsonToBase64(JSON.stringify(formObj));
    // const jsonPayload = new Blob([jsonBase64], { type: "application/json" });

    const formData = new FormData();
    formData.append("json", new Blob([JSON.stringify(formObj)], { type: "application/json" }), "inventory.json");

    if (fileInputUpdate && fileInputUpdate.files && fileInputUpdate.files.length > 0) {
			const fileList = Array.from(fileInputUpdate.files);
      for (const file of fileList) {
        if (!file) continue;
				let fileName: string = file.name || '';
        if (file.size > 64 * 1024 * 1024) { // 64 MB limit per file
          throw new Error(`File ${fileName} exceeds the maximum allowed size of 64 MB`);
        }
        if (fileName.length > 100) { // 100 characters limit for file name
          throw new Error(`File name ${fileName} exceeds the maximum allowed length of 100 characters`);
        }
        if (!allowedFileNameRegex.test(fileName)) {
          throw new Error(`File name ${fileName} contains invalid characters`);
        }
        if (!allowedFileExtensions.some(ext => fileName.toLowerCase().endsWith(ext))) {
          throw new Error(`File name ${fileName} has a forbidden extension`);
        }
				if (fileName.endsWith(".jfif")) {
					fileName = fileName.replace(/\.jfif$/i, ".jpeg");
				}
        formData.append("inventory-file-input", file, fileName);
      }
    }

    const response = await fetch("/api/update_inventory", {
      method: "POST",
      headers: {
        "credentials": "include"
      },
      body: formData
    });

    if (!response.ok) {
      throw new Error("Server returned an error: " + response.status + " " + response.statusText);
    }

    const data = await response.text();
		const returnedJson = JSON.parse(data);
		if (!returnedJson || !returnedJson.tagnumber) {
			throw new Error("Invalid response from server after inventory update");
		}
		if (fileInputUpdate) fileInputUpdate.value = "";
		clientLookupWarningMessage.style.display = "none";
		lastUpdateTime.textContent = "";
    await populateLocationForm(returnedJson.tagnumber, undefined);
    await renderInventoryTable();
  } catch (error) {
    console.error("Error updating inventory:", error);
    const errorMessage = error instanceof Error ? error.message : String(error);
		alert("Error updating inventory: " + errorMessage);
  } finally {
    updatingInventory = false;
    submitUpdate.disabled = false;
		showInventoryUpdateChanges();
  }
});

csvDownloadButton.addEventListener('click', async (event) => {
  event.preventDefault();
  csvDownloadButton.disabled = true;
  csvDownloadButton.textContent = 'Preparing download...';
  try {
    await fetchFilteredInventoryData(true); // true means CSV download
    await initializeInventoryPage();
  } catch (error) {
    console.error("Error downloading CSV:", error);
  } finally {
    csvDownloadButton.disabled = false;
    csvDownloadButton.textContent = 'Download Results';
  }
});

clientLookupReset.addEventListener("click", (event) => {
	event.preventDefault();
	resetInventoryLookupAndUpdateForm();
	updateURLFromFilters();
});

clientMoreDetails.addEventListener("click", (event) => {
  event.preventDefault();
  const tag = clientLookupTagInput.value;
  if (tag) {
    const url = new URL(window.location.origin + '/client');
		// const url = new URL(window.location.origin + '/client_images');
		url.searchParams.set('tagnumber', tag);
		window.open(url, '_blank');
  }
});

clientViewPhotos.addEventListener("click", (event) => {
	event.preventDefault();
	const tag = clientLookupTagInput.value;
	if (tag) {
		const url = new URL(window.location.origin + '/client_images');
		url.searchParams.set('tagnumber', tag);
		window.open(url, '_blank');
	}
});

clientAddPhotos.addEventListener("click", (event) => {
	event.preventDefault();
	fileInputUpdate.click();
});

clientStatusUpdate.addEventListener("change", async () => {
	await updateCheckoutStatus();
});

clientLookupForm.addEventListener("submit", async (event) => {
	event.preventDefault();
	await submitInventoryLookup();
	await updateCheckoutStatus();
});

document.addEventListener("DOMContentLoaded", async () => {
	updateFormContainerDisplay();
	window.addEventListener("resize", () => {
		updateFormContainerDisplay();
	});
	
	try {
		await initializeInventoryPage();
	} catch (e) {
		const errorMessage = e instanceof Error ? e.message : String(e);
		console.error("Error during inventory page initialization:", errorMessage);
	}
	if (Array.isArray(window.allTags)) {
		renderTagOptions(window.allTags);
	}

	document.addEventListener('tags:loaded', (event: CustomEvent<{ tags: number[] }>) => {
		const tags = (event && event.detail && Array.isArray(event.detail.tags)) ? event.detail.tags : window.allTags;
		renderTagOptions(tags || []);
	});
});

locationEl.addEventListener("keyup", async () => {
	const allLocations = await fetchAllLocations();
	const searchResults = getSortedLocations(locationEl, allLocations);
	const dataListElement = document.getElementById('location-suggestions') as HTMLDataListElement;
	dataListElement.innerHTML = '';
	searchResults.forEach(item => {
		const option = document.createElement('option');
		option.value = item.location;
		if (item.location_formatted) {
			option.label = item.location_formatted;
		}
		dataListElement.appendChild(option);
	});
});

function updateFormContainerDisplay() {
	if (window.matchMedia("(max-width: 768px)").matches) {
		updateFormContainer.classList.remove("grid-container", "inventory", "inventory-update-form");
		updateFormContainer.classList.add("flex-container", "horizontal");
	} else {
		updateFormContainer.classList.remove("flex-container", "horizontal");
		updateFormContainer.classList.add("grid-container", "inventory", "inventory-update-form");
	}
}