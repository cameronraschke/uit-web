let updatingInventory = false;

type InventoryForm = {
	last_update_time: string | null;
	tagnumber: number | null;
	system_serial: string | null;
	location: string | null;
	building: string | null;
	room: string | null;
	system_manufacturer: string | null;
	system_model: string | null;
	property_custodian: string | null;
	department_name: string | null;
	ad_domain: string | null;
	is_broken: boolean | null;
	disk_removed: boolean | null;
	status: string | null;
	acquired_date: string | null;
	note: string | null;
};

type Department = {
	department_name: string;
	department_name_formatted: string;
	department_sort_order: number;
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
};

type AllLocationsCache = {
	timestamp: number;
	locations: AllLocations[];
};

// Inventory lookup form elements
const inventoryFormContainer = document.getElementById('inventory-form-container') as HTMLElement;
const inventoryLookupWarningMessage = document.getElementById('existing-inventory-message') as HTMLElement;
const inventoryLookupForm = document.getElementById('inventory-lookup-form') as HTMLFormElement;
const inventoryLookupTagInput = document.getElementById('inventory-tag-lookup') as HTMLInputElement;
const inventoryLookupSystemSerialInput = document.getElementById('inventory-serial-lookup') as HTMLInputElement;
const inventoryLookupFormSubmitButton = document.getElementById('inventory-lookup-submit-button') as HTMLButtonElement;
const inventoryLookupFormResetButton = document.getElementById('inventory-lookup-reset-button') as HTMLButtonElement;
const inventoryLookupMoreDetailsButton = document.getElementById('inventory-lookup-more-details') as HTMLButtonElement;
const allTagsDatalist = document.getElementById('inventory-tag-suggestions') as HTMLDataListElement;
const clientImagesLink = document.getElementById('client_images_link') as HTMLAnchorElement;
const inventoryUpdateDomainSelect = document.getElementById('ad_domain') as HTMLSelectElement;
const inventorySearchDepartmentSelect = document.getElementById('inventory-search-department') as HTMLSelectElement;
const inventorySearchDomainSelect = document.getElementById('inventory-search-domain') as HTMLSelectElement;
const inventorySearchStatus = document.getElementById('inventory-search-status') as HTMLSelectElement;
const csvDownloadButton = document.getElementById('inventory-search-download-button') as HTMLButtonElement;
const printCheckoutAnchor = document.getElementById('print-checkout-link') as HTMLElement;

// Inventory update form elements
const inventoryUpdateForm = document.getElementById('inventory-update-form') as HTMLFormElement;
const inventoryUpdateFormSection = document.getElementById('inventory-update-section') as HTMLElement;
const lastUpdateTimeMessage = document.getElementById('last-update-time-message') as HTMLElement;
const inventoryUpdateLocationInput = document.getElementById('location') as HTMLInputElement;
const inventoryUpdateDepartmentSelect = document.getElementById('department_name') as HTMLSelectElement;
const inventoryUpdateFormSubmitButton = document.getElementById('inventory-update-submit-button') as HTMLButtonElement;
const inventoryUpdateFormCancelButton = document.getElementById('inventory-update-cancel-button') as HTMLButtonElement;
const building = inventoryUpdateForm.querySelector("#building") as HTMLInputElement;
const room = inventoryUpdateForm.querySelector("#room") as HTMLInputElement;
const systemManufacturer = inventoryUpdateForm.querySelector("#system_manufacturer") as HTMLInputElement;
const systemModel = inventoryUpdateForm.querySelector("#system_model") as HTMLInputElement;
const propertyCustodian = inventoryUpdateForm.querySelector("#property_custodian") as HTMLInputElement;
const acquiredDateInput = inventoryUpdateForm.querySelector("#acquired_date") as HTMLInputElement;
const isBroken = inventoryUpdateForm.querySelector("#is_broken") as HTMLSelectElement;
const diskRemoved = inventoryUpdateForm.querySelector("#disk_removed") as HTMLSelectElement;
const clientStatus = inventoryUpdateForm.querySelector("#status") as HTMLSelectElement;
const noteInput = inventoryUpdateForm.querySelector("#note") as HTMLInputElement;
const fileInput = inventoryUpdateForm.querySelector("#inventory-file-input") as HTMLInputElement;

const allowedFileNameRegex = /^[a-zA-Z0-9.\-_ ()]+\.[a-zA-Z]+$/; // file name + extension
const allowedFileExtensions = [".jpg", ".jpeg", ".jfif", ".png"];

const statusesThatIndicateBroken = ["needs-repair"];
const statusesThatIndicateCheckout = ["checked-out", "reserved-for-checkout"];

const allInventoryUpdateFields = [
	inventoryLookupTagInput,
	inventoryLookupSystemSerialInput,
	inventoryUpdateLocationInput,
	building,
	room,
	systemManufacturer,
	systemModel,
	inventoryUpdateDepartmentSelect,
	inventoryUpdateDomainSelect,
	propertyCustodian,
	acquiredDateInput,
	isBroken,
	diskRemoved,
	clientStatus,
	fileInput,
	noteInput,
];

const requiredInventoryUpdateFields = [
	inventoryLookupTagInput,
	inventoryLookupSystemSerialInput,
	inventoryUpdateLocationInput,
	inventoryUpdateDepartmentSelect,
	inventoryUpdateDomainSelect,
	clientStatus
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

function getLocationSearchResults(inputElement: HTMLInputElement, data: Array<AllLocations>): Array<{ location: string, location_formatted: string | null }> {
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
	const lookupTag: number | null = inventoryLookupTagInput.value ? Number(inventoryLookupTagInput.value) : (searchParams.get('tagnumber') ? Number(searchParams.get('tagnumber')) : null);
  const lookupSerial: string | null = inventoryLookupSystemSerialInput.value || searchParams.get('system_serial') || null;

  if (!lookupTag && !lookupSerial) {
    inventoryLookupWarningMessage.style.display = "block";
    inventoryLookupWarningMessage.textContent = "Please provide a tag number or serial number to look up.";
    return;
  }
  if (lookupTag && isNaN(Number(lookupTag))) {
    inventoryLookupWarningMessage.style.display = "block";
    inventoryLookupWarningMessage.textContent = "Tag number must be numeric.";
    return;
  }
  if (lookupSerial && (lookupSerial.length < 4 || lookupSerial.length > 20)) {
    inventoryLookupWarningMessage.style.display = "block";
    inventoryLookupWarningMessage.textContent = "Serial number must be between 4 and 20 characters long.";
    return;
  }
  if (lookupTag && lookupTag.toString().length != 6) {
    inventoryLookupWarningMessage.style.display = "block";
    inventoryLookupWarningMessage.textContent = "Tag number must be exactly 6 digits long.";
    return;
  }

	inventoryLookupFormResetButton.style.display = "inline-block";
	inventoryLookupMoreDetailsButton.style.display = "inline-block";
	inventoryLookupMoreDetailsButton.disabled = false;


	try {
		const lookupResult: ClientLookupResult | null = await lookupTagOrSerial(lookupTag, lookupSerial);
		if (!lookupResult) {
			inventoryLookupWarningMessage.style.display = "block";
			inventoryLookupWarningMessage.textContent = "No inventory entry was found for the provided tag number or serial number. A new entry can be created.";
			if (!inventoryLookupTagInput.value) inventoryLookupTagInput.focus();
			else if (!inventoryLookupSystemSerialInput.value) inventoryLookupSystemSerialInput.focus();
		}

		if (lookupResult) {
			if (lookupResult.tagnumber && !isNaN(Number(lookupResult.tagnumber))) {
				searchParams.set("tagnumber", lookupResult.tagnumber ? lookupResult.tagnumber.toString() : '');
				inventoryLookupTagInput.value = Number(lookupResult.tagnumber).toString();
				inventoryLookupTagInput.style.backgroundColor = "gainsboro";
				inventoryLookupTagInput.readOnly = true;
				clientImagesLink.href = `/client_images?tagnumber=${lookupResult.tagnumber}`;
				clientImagesLink.target = "_blank";
				clientImagesLink.style.display = "inline";
			}
			if (lookupResult.system_serial && lookupResult.system_serial && lookupResult.system_serial.trim().length > 0) {
				searchParams.set("system_serial", lookupResult.system_serial ? lookupResult.system_serial.trim() : '');
				inventoryLookupSystemSerialInput.value = lookupResult.system_serial.trim();
				inventoryLookupSystemSerialInput.value = lookupResult.system_serial ? lookupResult.system_serial : "";
				inventoryLookupSystemSerialInput.style.backgroundColor = "gainsboro";
				inventoryLookupSystemSerialInput.readOnly = true;
			}

			inventoryLookupFormSubmitButton.disabled = true;
			inventoryLookupFormSubmitButton.style.cursor = "not-allowed";
			inventoryLookupFormSubmitButton.style.border = "1px solid gray";
			inventoryLookupMoreDetailsButton.style.display = "inline-block";

			inventoryLookupMoreDetailsButton.disabled = false;
			inventoryLookupMoreDetailsButton.style.cursor = "pointer";

			if (lookupResult.tagnumber || lookupResult.system_serial) {
				await populateLocationForm(lookupResult.tagnumber ? lookupResult.tagnumber : undefined, lookupResult.system_serial ? lookupResult.system_serial : undefined);
			}
		} else {
			inventoryLookupMoreDetailsButton.disabled = true;
			inventoryLookupMoreDetailsButton.style.cursor = "not-allowed";
			const tagNum = inventoryLookupTagInput.value ? Number(inventoryLookupTagInput.value) : '';
			const serialNum = inventoryLookupSystemSerialInput.value ? inventoryLookupSystemSerialInput.value : '';
			searchParams.set("tagnumber", tagNum.toString());
			searchParams.set("system_serial", serialNum);
			await populateLocationForm(tagNum ? tagNum : undefined, serialNum ? serialNum : undefined);
		}
	} catch (error) {
		const errorMessage = error instanceof Error ? error.message : String(error);
		inventoryLookupWarningMessage.style.display = "block";
		inventoryLookupWarningMessage.textContent = "Error looking up inventory entry: " + errorMessage;
	} finally {
		// Set 'update' parameter in URL
		searchParams.set('update', 'true');
		history.replaceState(null, '', window.location.pathname + '?' + searchParams.toString());
	}
}

async function updateCheckoutStatus() {
	if (statusesThatIndicateCheckout.includes(clientStatus.value)) {
		printCheckoutAnchor.setAttribute('href', `/checkout-form?tagnumber=${encodeURIComponent(inventoryLookupTagInput.value)}`);
		printCheckoutAnchor.setAttribute('target', '_blank');
		printCheckoutAnchor.textContent = 'Print Checkout Form';
	} else {
		if (printCheckoutAnchor) {
			printCheckoutAnchor.innerHTML = '';
		}
	}
}

function resetInventoryLookupAndUpdateForm() {
	inventoryLookupForm.reset();
	inventoryUpdateForm.reset();
	for (const el of allInventoryUpdateFields) {
		if (el instanceof HTMLInputElement) {
			resetInputElement(el, "", false, undefined);
		}
		if (el instanceof HTMLSelectElement) {
			resetSelectElement(el, "", false, undefined);
		}
	}
	inventoryLookupTagInput.placeholder = "Enter Tag Number";
	inventoryLookupSystemSerialInput.placeholder = "Enter System Serial";
	inventoryLookupFormSubmitButton.style.cursor = "pointer";

	inventoryLookupFormSubmitButton.style.border = "1px solid black";
	inventoryLookupFormSubmitButton.disabled = false;
	inventoryLookupFormResetButton.style.display = "none";
	inventoryLookupMoreDetailsButton.style.display = "none";
	inventoryLookupMoreDetailsButton.disabled = false;
	inventoryLookupMoreDetailsButton.style.cursor = "pointer";
	inventoryUpdateFormSection.style.display = "none";
	inventoryLookupWarningMessage.style.display = "none";
	inventoryLookupWarningMessage.textContent = "";
	lastUpdateTimeMessage.textContent = "";
	inventoryLookupTagInput.focus();
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
	const tagNum = tag ? tag : inventoryLookupTagInput.value ? Number(inventoryLookupTagInput.value) : null;
	const serialNum = serial ? serial : inventoryLookupSystemSerialInput.value ? String(inventoryLookupSystemSerialInput.value) : null;
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
	const inputs = inventoryUpdateForm.querySelectorAll("input, select, textarea");

	inputs.forEach((el: HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement) => {
		el.dataset.initialValue = el.value;

		const handleInputUpdate = () => {
			if (updatingInventory) return;

			// Check if value matches the initial value
			if (el.dataset.initialValue === el.value) {
				el.classList.remove("changed-input");
				return;
			}

			// If different, update classes
			for (const cssClass of inputCSSClasses) {
				el.classList.remove(cssClass);
			}
			el.classList.add("changed-input");
		};

		el.oninput = handleInputUpdate;
		el.onchange = handleInputUpdate;
	});
}

async function populateLocationForm(tag?: number, serial?: string): Promise<void> {
	for (const el of requiredInventoryUpdateFields) {
		el.required = true;
	}

	// reset/zero/clear out all fields before processing new data
	resetInputElement(inventoryLookupTagInput, "Enter Tag Number", false, undefined);
	if (inventoryLookupTagInput.value) {
		inventoryLookupTagInput.readOnly = true;
		inventoryLookupTagInput.value = inventoryLookupTagInput.value.toString().trim();
		inventoryLookupTagInput.classList.add("readonly-input");
	} else {
		inventoryLookupTagInput.classList.add("empty-required-input");
	}

	resetInputElement(inventoryLookupSystemSerialInput, "Enter System Serial", false, undefined);
	if (inventoryLookupSystemSerialInput.value) {
		inventoryLookupSystemSerialInput.readOnly = true;
		inventoryLookupSystemSerialInput.value = inventoryLookupSystemSerialInput.value.trim();
		inventoryLookupSystemSerialInput.classList.add("readonly-input");
	} else {
		inventoryLookupSystemSerialInput.classList.add("empty-required-input");
	}

	resetInputElement(inventoryUpdateLocationInput, "Enter Location", false, "empty-required-input");

	resetInputElement(building, "Building", false, "empty-input");

	resetInputElement(room, "Room", false, "empty-input");

	resetInputElement(systemManufacturer, "System Manufacturer", false, "empty-input");

	resetInputElement(systemModel, "System Model", false, "empty-input");

	resetSelectElement(inventoryUpdateDepartmentSelect, "Select Department", false, "empty-required-input");
	try { 
		await populateDepartmentSelect(inventoryUpdateDepartmentSelect)
	} catch(e) {
		const errorMessage = e instanceof Error ? e.message : String(e);
		console.error(`Could not fetch all departments: ${errorMessage}`)
	}

	resetSelectElement(inventoryUpdateDomainSelect, "Select Domain", false, "empty-required-input");
	try { 
		await populateDomainSelect(inventoryUpdateDomainSelect);
	} catch(e) {
		const errorMessage = e instanceof Error ? e.message : String(e);
		console.error(`Could not fetch all domains: ${errorMessage}`)
	}

	resetInputElement(propertyCustodian, "Property Custodian", false, "empty-input");

	resetInputElement(acquiredDateInput, "Acquired Date", false, "empty-input");

	resetSelectElement(isBroken, "Is Broken?", false, "empty-required-input");
		if (isBroken) {
		const op1 = document.createElement("option");
		op1.value = "true";
		op1.textContent = "Is broken"

		const op2 = document.createElement("option");
		op2.value = "false";
		op2.textContent = "Is functional";

		const op3 = document.createElement("option");
		op3.value = "unknown";
		op3.textContent = "Unknown";

		isBroken.append(op1);
		isBroken.append(op2);
		isBroken.append(op3);
	}

	resetSelectElement(diskRemoved, "Disk Removed?", false, "empty-input");
	if (diskRemoved) {
		const op1 = document.createElement("option");
		op1.value = "true";
		op1.textContent = "Yes, disk removed"

		const op2 = document.createElement("option");
		op2.value = "false";
		op2.textContent = "No, disk present";

		const op3 = document.createElement("option");
		op3.value = "unknown";
		op3.textContent = "Unknown";

		diskRemoved.append(op1);
		diskRemoved.append(op2);
		diskRemoved.append(op3);
	}

	resetSelectElement(clientStatus, "Select Client Status", false, "empty-required-input");
	try { 
		await populateStatusSelect(clientStatus);
	} catch(e) {
		const errorMessage = e instanceof Error ? e.message : String(e);
		console.error(`Could not fetch all statuses: ${errorMessage}`)
	}

	resetInputElement(fileInput, "", false, undefined);

	resetInputElement(noteInput, "Enter Note", false, "empty-input");

	lastUpdateTimeMessage.style.display = "none";
	try {
		const locationFormData = await getLocationFormData(tag, serial);
		if (locationFormData) {
			if (locationFormData.last_update_time) {
				const lastUpdate = new Date(locationFormData.last_update_time);
				if (isNaN(lastUpdate.getTime())) {
					lastUpdateTimeMessage.textContent = 'Unknown timestamp of last update';
				} else {
					lastUpdateTimeMessage.textContent = `Last updated: ${lastUpdate.toLocaleString()}` || '';
				}
				lastUpdateTimeMessage.style.display = "block";
			}
		
			if (locationFormData.tagnumber) {
				inventoryLookupTagInput.value = locationFormData.tagnumber.toString();
				inventoryLookupTagInput.classList.remove("empty-required-input");
				inventoryLookupTagInput.classList.add("readonly-input");
			} else {
				inventoryLookupTagInput.classList.add("empty-required-input");
				inventoryLookupTagInput.focus();
			}

			if (locationFormData.system_serial) {
				inventoryLookupSystemSerialInput.value = locationFormData.system_serial.trim();
				inventoryLookupSystemSerialInput.classList.remove("empty-required-input");
				inventoryLookupSystemSerialInput.classList.add("readonly-input");
			} else {
				inventoryLookupSystemSerialInput.classList.add("empty-required-input");
				inventoryLookupSystemSerialInput.focus();
			}

			if (locationFormData.location ) {
				inventoryUpdateLocationInput.value = locationFormData.location.trim();
				inventoryUpdateLocationInput.classList.remove("empty-required-input");
			} else {
				inventoryUpdateLocationInput.classList.add("empty-required-input");
			}

			if (locationFormData.building) {
				building.value = locationFormData.building.trim();
				building.classList.remove("empty-input");
			} else {
				building.classList.add("empty-input");
			}

			if (locationFormData.room) {
				room.value = locationFormData.room.trim();
				room.classList.remove("empty-input");
			} else {
				room.classList.add("empty-input");
			}

			if (locationFormData.system_manufacturer) {
				systemManufacturer.readOnly = true;
				systemManufacturer.value = locationFormData.system_manufacturer.trim();
				systemManufacturer.classList.remove("empty-input");
				systemManufacturer.classList.add("readonly-input");
			} else {
				systemManufacturer.classList.add("empty-input");
			}

			if (locationFormData.system_model) {
				systemModel.readOnly = true;
				systemModel.value = locationFormData.system_model.trim();
				systemModel.classList.remove("empty-input");
				systemModel.classList.add("readonly-input");
			} else {
				systemModel.classList.add("empty-input");
			}

			if (locationFormData.department_name) {
				inventoryUpdateDepartmentSelect.value = locationFormData.department_name.trim();
				inventoryUpdateDepartmentSelect.classList.remove("empty-required-input");
			} else {
				inventoryUpdateDepartmentSelect.classList.add("empty-required-input");
			}

			if (locationFormData.ad_domain) {
				inventoryUpdateDomainSelect.value = locationFormData.ad_domain.trim();
				inventoryUpdateDomainSelect.classList.remove("empty-required-input");
			} else {
				inventoryUpdateDomainSelect.classList.add("empty-required-input");
			}

			if (locationFormData.property_custodian) {
				propertyCustodian.value = locationFormData.property_custodian.trim();
				propertyCustodian.classList.remove("empty-input");
			} else {
				propertyCustodian.classList.add("empty-input");
			}

			if (locationFormData.acquired_date) {
				const acquiredDateValue = locationFormData.acquired_date ? new Date(locationFormData.acquired_date) : null;
				if (acquiredDateValue && !isNaN(acquiredDateValue.getTime())) {
					// Format as YYYY-MM-DD for input[type="date"]
					const year = acquiredDateValue.getFullYear();
					const month = String(acquiredDateValue.getMonth() + 1).padStart(2, '0');
					const day = String(acquiredDateValue.getDate()).padStart(2, '0');
					const acquiredDateFormatted = `${year}-${month}-${day}`;
					acquiredDateInput.value = acquiredDateFormatted;
					acquiredDateInput.classList.remove("empty-input");
				}
			} else {
				acquiredDateInput.classList.add("empty-input");
			}
			
			if (locationFormData.is_broken === true) {
				isBroken.value = "true";
				isBroken.classList.remove("empty-input");
			} else if (locationFormData.is_broken === false) {
				isBroken.value = "false";
				isBroken.classList.remove("empty-input");
			} else {
				isBroken.classList.add("empty-input");
			}

			if (locationFormData.disk_removed === true) {
				diskRemoved.value = "true";
				diskRemoved.classList.remove("empty-input");
			} else if (locationFormData.disk_removed === false) {
				diskRemoved.value = "false";
				diskRemoved.classList.remove("empty-input");
			} else {
				diskRemoved.classList.add("empty-input");
			}

			if (locationFormData.status) {
				clientStatus.value = locationFormData.status.trim();
				clientStatus.classList.remove("empty-required-input");
			} else {
				clientStatus.classList.add("empty-required-input");
			}

			if (locationFormData.note) {
				noteInput.value = locationFormData.note.trim();
				noteInput.classList.remove("empty-input");
			} else {
				noteInput.classList.add("empty-input");
			}
		}
		await updateCheckoutStatus();
	} catch (error) {
		const errorMessage = error instanceof Error ? error.message : String(error);
		console.error("Error populating location form:", errorMessage);
	} finally {
		showInventoryUpdateChanges();
		inventoryUpdateFormSection.style.display = "block";
		inventoryUpdateLocationInput.focus();
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

async function initializeInventoryPage() {
	initializeSearch();

	try {
		const tasks = [
			populateManufacturerSelect(true),
			populateModelSelect(true),
			populateDomainSelect(inventorySearchDomainSelect, true),
			populateDepartmentSelect(inventorySearchDepartmentSelect, true),
			populateStatusSelect(inventorySearchStatus),
			fetchFilteredInventoryData()
		];
		await Promise.all(tasks);

		// Check URL parameters for auto lookup
		const urlParams = new URLSearchParams(window.location.search);
		const updateParam: string | null = urlParams.get('update');
		const tagnumberParam: string | null = urlParams.get('tagnumber');
		if (tagnumberParam && updateParam === 'true') {
			inventoryLookupTagInput.value = tagnumberParam;
			await submitInventoryLookup();
		}
	} catch (e) {
		const errorMessage = e instanceof Error ? e.message : String(e);
		console.error("Error initializing inventory page:", errorMessage);
	}
}

inventoryUpdateFormCancelButton.addEventListener("click", (event) => {
  event.preventDefault();
	history.replaceState(null, '', window.location.pathname);
  resetInventoryLookupAndUpdateForm();
	setURLParameter(null, null);
});

inventoryLookupTagInput.addEventListener("keyup", (event: KeyboardEvent) => {
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

inventoryUpdateForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  inventoryUpdateFormSubmitButton.disabled = true;
  if (updatingInventory) return;
  updatingInventory = true;

	updateURLFromFilters();

  try {
		const jsonObject = {} as InventoryForm;
    jsonObject.tagnumber = inventoryLookupTagInput && inventoryLookupTagInput.value ? Number(inventoryLookupTagInput.value) : null;
    jsonObject.system_serial = inventoryLookupSystemSerialInput && inventoryLookupSystemSerialInput.value ? String(inventoryLookupSystemSerialInput.value) : null;
    if (!inventoryLookupTagInput && !inventoryLookupSystemSerialInput) {
      throw new Error("No tag or serial input fields found in DOM");
    }
    const getInputValue = (documentID: string): string | null => {
      const input = inventoryUpdateForm.querySelector(documentID) as HTMLInputElement | null;
      return input && input.value ? String(input.value) : null;
    };
		const locationValue = getInputValue("#location");
		if (!locationValue || locationValue.trim().length === 0) {
			throw new Error("Location field cannot be empty");
		}
    jsonObject.location = locationValue;
		
		jsonObject.building = getInputValue("#building");
		jsonObject.room = getInputValue("#room");
    jsonObject.system_manufacturer = getInputValue("#system_manufacturer");
    jsonObject.system_model = getInputValue("#system_model");
		jsonObject.property_custodian = getInputValue("#property_custodian");

		const departmentValue = getInputValue("#department_name");
		if (!departmentValue || departmentValue.trim().length === 0) {
			throw new Error("Department field cannot be empty");
		}
    jsonObject.department_name = departmentValue;

    jsonObject.ad_domain = getInputValue("#ad_domain");
    const brokenBool = getInputValue("#is_broken");
      if (brokenBool === "true") jsonObject.is_broken = true;
      else if (brokenBool === "false") jsonObject.is_broken = false;
      else jsonObject.is_broken = null;
		const diskRemovedBool = getInputValue("#disk_removed");
			if (diskRemovedBool === "true") jsonObject.disk_removed = true;
			else if (diskRemovedBool === "false") jsonObject.disk_removed = false;
			else jsonObject.disk_removed = null;

		const statusValue = getInputValue("#status");
		if (!statusValue || statusValue.trim().length === 0) {
			throw new Error("Status field cannot be empty");
		}
    jsonObject.status = statusValue;
		if (getInputValue("#acquired_date")) {
			jsonObject.acquired_date = new Date((getInputValue("#acquired_date") as string) + "T00:00:00").toISOString() || null;
		}
    jsonObject.note = getInputValue("#note");
    // const jsonBase64 = jsonToBase64(JSON.stringify(jsonObject));
    // const jsonPayload = new Blob([jsonBase64], { type: "application/json" });

    const formData = new FormData();
    formData.append("json", new Blob([JSON.stringify(jsonObject)], { type: "application/json" }), "inventory.json");

    if (fileInput && fileInput.files && fileInput.files.length > 0) {
			const fileList = Array.from(fileInput.files);
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
		if (fileInput) fileInput.value = "";
		inventoryLookupWarningMessage.style.display = "none";
		lastUpdateTimeMessage.textContent = "";
    await populateLocationForm(returnedJson.tagnumber, undefined);
    await fetchFilteredInventoryData();
  } catch (error) {
    console.error("Error updating inventory:", error);
    const errorMessage = error instanceof Error ? error.message : String(error);
		alert("Error updating inventory: " + errorMessage);
  } finally {
    updatingInventory = false;
    inventoryUpdateFormSubmitButton.disabled = false;
		showInventoryUpdateChanges();
  }
});

csvDownloadButton.addEventListener('click', async (event) => {
  event.preventDefault();
  csvDownloadButton.disabled = true;
  csvDownloadButton.textContent = 'Preparing download...';
  try {
    await fetchFilteredInventoryData(true);
  } finally {
    await initializeInventoryPage();
    csvDownloadButton.disabled = false;
    csvDownloadButton.textContent = 'Download Results';
  }
});

inventoryLookupFormResetButton.addEventListener("click", (event) => {
  event.preventDefault();
	history.replaceState(null, '', window.location.pathname);
  resetInventoryLookupAndUpdateForm();
	setURLParameter(null, null);
});

inventoryLookupMoreDetailsButton.addEventListener("click", (event) => {
  event.preventDefault();
  const tag = inventoryLookupTagInput.value;
  if (tag) {
    // const url = `/client?tagnumber=${encodeURIComponent(tag)}`;
		const url = new URL(window.location.origin + '/client_images');
		url.searchParams.set('tagnumber', tag);
		window.open(url, '_blank');
  }
});

clientStatus.addEventListener("change", async () => {
	await updateCheckoutStatus();
});

inventoryLookupForm.addEventListener("submit", async (event) => {
	event.preventDefault();
	await submitInventoryLookup();
	await updateCheckoutStatus();
});

document.addEventListener("DOMContentLoaded", async () => {
	updateFormContainerDisplay();
	window.addEventListener("resize", () => {
		updateFormContainerDisplay();
	});
  await initializeInventoryPage();
	updateFiltersFromURL();
	if (Array.isArray(window.allTags)) {
		renderTagOptions(window.allTags);
	}

	document.addEventListener('tags:loaded', (event: CustomEvent<{ tags: number[] }>) => {
		const tags = (event && event.detail && Array.isArray(event.detail.tags)) ? event.detail.tags : window.allTags;
		renderTagOptions(tags || []);
	});
});

inventoryUpdateLocationInput.addEventListener("keyup", async () => {
	const allLocations = await fetchAllLocations();
	const searchResults = getLocationSearchResults(inventoryUpdateLocationInput, allLocations);
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
		inventoryFormContainer.classList.remove("grid-container", "inventory", "inventory-update-form");
		inventoryFormContainer.classList.add("flex-container", "horizontal");
	} else {
		inventoryFormContainer.classList.remove("flex-container", "horizontal");
		inventoryFormContainer.classList.add("grid-container", "inventory", "inventory-update-form");
	}
}