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
		const data: AllLocations[] = await fetchData('/api/overview/all_locations', false);
		if (!data || !Array.isArray(data)) {
			throw new Error("No data returned from /api/overview/all_locations");
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
   if (tagnumber === null && (serial === null || serial.trim() === "")) {
    console.log("No tag or serial provided");
    return null;
  }
  try {
		const query = new URLSearchParams();
		if (tagnumber !== null && validateTagInput(tagnumber)) {
			query.append("tagnumber", tagnumber.toString());
		} else if (serial !== null) {
			query.append("system_serial", serial);
		}
    const data = await fetchData(`/api/client/lookup?${query.toString()}`);
    if (!data) {
      console.log("No data returned from /api/client/lookup");
			return null;
    }
		const jsonResponse: ClientLookupResult = data as ClientLookupResult;
		if (!jsonResponse || (jsonResponse.tagnumber === null && jsonResponse.system_serial === null)) {
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
	updateURLFromAdvFilters();
	const searchParams: URLSearchParams = new URLSearchParams(window.location.search);
	const lookupTag: number | null = clientLookupTagInput.value ? Number(clientLookupTagInput.value) : (searchParams.get('tagnumber') ? Number(searchParams.get('tagnumber')) : null);
  const lookupSerial: string | null = clientLookupSerial.value || searchParams.get('system_serial') || null;

	clientLookupTagInput.dataset.initialValue = clientLookupTagInput.value;
	clientLookupSerial.dataset.initialValue = clientLookupSerial.value;

  if (!validateTagInput(lookupTag) && !validateSerialInput(lookupSerial)) {
    clientLookupWarningMessage.style.display = "block";
    clientLookupWarningMessage.textContent = "Please provide a valid tag number or serial number to look up.";
		setURLParameter('tagnumber', null);
		setURLParameter('system_serial', null);
		setURLParameter('update', null);
    return;
  }

	for (const btn of buttonsVisibleWhenUpdating) {
		btn.style.display = "inline-block";
		btn.disabled = false;
	}

	try {
		const lookupResult: ClientLookupResult | null = await lookupTagOrSerial(lookupTag, lookupSerial);

		if (lookupResult !== null && (lookupResult.tagnumber !== null || (lookupResult.system_serial && lookupResult.system_serial.trim() !== ""))) {
			if (lookupResult.tagnumber && !isNaN(Number(lookupResult.tagnumber))) {
				if (validateTagInput(lookupResult.tagnumber)) {
					setURLParameter("tagnumber", lookupResult.tagnumber.toString());
				};
				clientLookupTagInput.value = Number(lookupResult.tagnumber).toString();
				clientLookupTagInput.dataset.initialValue = lookupResult.tagnumber ? lookupResult.tagnumber.toString() : "";
				clientLookupTagInput.readOnly = true;
			}
			if (lookupResult.system_serial && lookupResult.system_serial && lookupResult.system_serial.trim().length > 0) {
				if (validateSerialInput(lookupResult.system_serial)) {
					setURLParameter("system_serial", lookupResult.system_serial.trim());
				}
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

			if (validateTagInput(lookupResult.tagnumber) && validateSerialInput(lookupResult.system_serial)) {
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
		// Set 'update' parameter in URL
		searchParams.set('update', 'true');
		history.replaceState(null, '', window.location.pathname + '?' + searchParams.toString());
	} catch (error) {
		const errorMessage = error instanceof Error ? error.message : String(error);
		console.error("Error during inventory lookup: " + errorMessage);
		clientLookupWarningMessage.style.display = "block";
		clientLookupWarningMessage.textContent = "Error looking up inventory entry. Check console.";
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
	if (clientViewPhotosButton) clientViewPhotosButton.textContent = "View Photos";
	if (clientAddPhotosButton) clientAddPhotosButton.textContent = "Add Photos";
	clientLookupTagInput.placeholder = "Enter Tag Number";
	clientLookupSerial.placeholder = "Enter System Serial";

	clientLookupSubmitButton.disabled = false;
	clientLookupSubmitButton.style.cursor = "pointer";
	clientLookupSubmitButton.style.border = "1px solid black";
	clientLookupSubmitButton.style.backgroundColor = "";
	clientLookupSubmitButton.style.display = "inline-block";

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

async function getLocationFormData(tag?: number, serial?: string): Promise<InventoryForm | null> {
	const url = new URL('/api/client/location_form_data', window.location.origin);
	const tagNum = tag ? tag : clientLookupTagInput.value ? Number(clientLookupTagInput.value) : null;
	const serialNum = serial ? serial : clientLookupSerial.value ? String(clientLookupSerial.value) : null;
	if (tagNum === null && (serialNum === null || serialNum.trim() === "")) {
		console.log("No tag or serial provided for location form data");
		return null;
	}
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
	const inputs = updateForm.querySelectorAll("input, select, textarea") as NodeListOf<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>;

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
		clientLookupTagInput.tabIndex = -1;
	} else {
		clientLookupTagInput.classList.add("empty-required-input");
		clientLookupTagInput.classList.remove("readonly-input");
		clientLookupTagInput.removeAttribute("tabindex");
	}

	resetInputElement(clientLookupSerial, "Enter System Serial", false, undefined);
	clientLookupSerial.value = clientLookupSerial.dataset.initialValue || "";
	if (clientLookupSerial.value) {
		clientLookupSerial.readOnly = true;
		clientLookupSerial.value = clientLookupSerial.value.trim();
		clientLookupSerial.classList.add("readonly-input");
		clientLookupSerial.tabIndex = -1;
	} else {
		clientLookupSerial.classList.add("empty-required-input");
		clientLookupSerial.classList.remove("readonly-input");
		clientLookupSerial.removeAttribute("tabindex");
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

	resetSelectElement(adDomainUpdate, "Select AD Domain", false, "empty-required-input");
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

	resetSelectElement(isBrokenUpdate, "Is Functional?", false, "empty-input");
		if (isBrokenUpdate) {
		const op1 = document.createElement("option");
		op1.value = "false";
		op1.textContent = "Yes, it is functional";

		const op2 = document.createElement("option");
		op2.value = "true";
		op2.textContent = "No, it is broken";

		const op3 = document.createElement("option");
		op3.value = "unknown";
		op3.textContent = "Unknown";

		isBrokenUpdate.append(op1);
		isBrokenUpdate.append(op2);
		isBrokenUpdate.append(op3);
	}

	resetSelectElement(diskRemovedUpdate, "Disk Present?", false, "empty-input");
	if (diskRemovedUpdate) {
		const op1 = document.createElement("option");
		op1.value = "false";
		op1.textContent = "Yes, disk present";

		const op2 = document.createElement("option");
		op2.value = "true";
		op2.textContent = "No, disk removed"

		const op3 = document.createElement("option");
		op3.value = "unknown";
		op3.textContent = "Unknown";

		diskRemovedUpdate.append(op1);
		diskRemovedUpdate.append(op2);
		diskRemovedUpdate.append(op3);
	}

	removeCSSClasses(lastHardwareCheckUpdate);
	lastHardwareCheckUpdate.classList.add("empty-input");
	lastHardwareCheckUpdate.value = "";

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
		
			if (locationFormData.file_count !== null) clientViewPhotosButton.textContent = `View Photos (${locationFormData.file_count})`;

			if (locationFormData.tagnumber) {
				clientLookupTagInput.value = locationFormData.tagnumber.toString();
				clientLookupTagInput.classList.remove("empty-required-input");
				clientLookupTagInput.classList.add("readonly-input");
				clientLookupTagInput.readOnly = true;
				clientLookupTagInput.tabIndex = -1;
			} else {
				clientLookupTagInput.classList.add("empty-required-input");
				clientLookupTagInput.classList.remove("readonly-input");
				clientLookupTagInput.removeAttribute("tabindex");
			}

			if (locationFormData.system_serial) {
				clientLookupSerial.value = locationFormData.system_serial.trim();
				clientLookupSerial.classList.remove("empty-required-input");
				clientLookupSerial.classList.add("readonly-input");
				clientLookupSerial.readOnly = true;
				clientLookupSerial.tabIndex = -1;
			} else {
				clientLookupSerial.classList.add("empty-required-input");
				clientLookupSerial.classList.remove("readonly-input");
				clientLookupSerial.removeAttribute("tabindex");
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
				manufacturerUpdate.tabIndex = -1;
			} else {
				manufacturerUpdate.classList.add("empty-input");
				manufacturerUpdate.classList.remove("readonly-input");
				manufacturerUpdate.removeAttribute("tabindex");
			}

			if (locationFormData.system_model) {
				modelUpdate.readOnly = true;
				modelUpdate.value = locationFormData.system_model.trim();
				modelUpdate.classList.remove("empty-input");
				modelUpdate.classList.add("readonly-input");
				modelUpdate.tabIndex = -1;
			} else {
				modelUpdate.classList.add("empty-input");
				modelUpdate.classList.remove("readonly-input");
				modelUpdate.removeAttribute("tabindex");
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

			if (locationFormData.retired_date) {
				const retiredDateValue = locationFormData.retired_date ? new Date(locationFormData.retired_date) : null;
				if (retiredDateValue && !isNaN(retiredDateValue.getTime())) {
					// Format as YYYY-MM-DD for input[type="date"]
					const year = retiredDateValue.getFullYear();
					const month = String(retiredDateValue.getMonth() + 1).padStart(2, '0');
					const day = String(retiredDateValue.getDate()).padStart(2, '0');
					const retiredDateFormatted = `${year}-${month}-${day}`;
					retiredDateUpdate.value = retiredDateFormatted;
					retiredDateUpdate.classList.remove("empty-input");
				}
			} else {
				retiredDateUpdate.classList.add("empty-input");
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

			if (locationFormData.last_hardware_check) {
				const hardwareCheckDate = new Date(locationFormData.last_hardware_check);
				const hardwareCheckDateLocal = !isNaN(hardwareCheckDate.getTime())
					? new Date(hardwareCheckDate.getTime() - hardwareCheckDate.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
					: "";
				if (hardwareCheckDateLocal && !isNaN(new Date(hardwareCheckDateLocal).getTime())) {
					lastHardwareCheckUpdate.value = hardwareCheckDateLocal;
					lastHardwareCheckUpdate.classList.remove("empty-input");
				} else {
					lastHardwareCheckUpdate.value = "";
					lastHardwareCheckUpdate.classList.add("empty-input");
				}
			} else {
				lastHardwareCheckUpdate.value = "";
				lastHardwareCheckUpdate.classList.add("empty-input");
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
		const data: Array<Department> = await fetchData('/api/overview/all_departments', false);
		if (!data || !Array.isArray(data) || data.length === 0) {
			throw new Error('No data returned from /api/overview/all_departments');
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
		const allDeviceTypes: DeviceType[] = await fetchData(`/api/overview/all_device_types`, false);
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
	if (!selectEl) {
		console.warn("Device type select element not found");
		return;
	}
	try {
		const deviceTypes = await fetchAllDeviceTypes(purgeCache);
		if (!deviceTypes || !Array.isArray(deviceTypes)) {
			throw new Error("Invalid device types data");
		}
		resetSelectElement(selectEl, "Device Type", false, undefined);
		const uniqueMetaCategores = new Set(deviceTypes.map(dev => dev.device_meta_category));
		const sortedUniqueMetaCategories = [...uniqueMetaCategores].sort((a, b) => {
			if (!a && !b) return 0;
			const aVal = a ? a.trim().toLowerCase() : '';
			const bVal = b ? b.trim().toLowerCase() : '';
			if (aVal && aVal === 'unknown/other') return 1;
			if (bVal && bVal === 'unknown/other') return -1;
			return aVal.localeCompare(bVal);
		});
		for (const device of sortedUniqueMetaCategories) {
			const orgEl = document.createElement('optgroup');
			orgEl.label = device ? device.trim() : 'N/A';
			selectEl.appendChild(orgEl);
		}

		const sortedDeviceTypes = [...deviceTypes].sort((a, b) => {
			const aVal = (a.device_type_formatted || "").trim().toLowerCase();
			const bVal = (b.device_type_formatted || "").trim().toLowerCase();
			return aVal.localeCompare(bVal);
		});
		for (const deviceType of sortedDeviceTypes) {
			if (!deviceType.device_type || !deviceType.device_type_formatted) {
				console.warn("Skipping invalid device type entry:", deviceType);
				continue;
			}
			const optionEl = document.createElement('option');
			optionEl.value = deviceType.device_type.trim();
			optionEl.textContent = (deviceType.device_type_formatted.trim()) + (deviceType.device_type_count !== null ? ` (${deviceType.device_type_count})` : '');
			const parentOptGroup = Array.from(selectEl.children).find(child => {
				return child instanceof HTMLOptGroupElement && child.label === (deviceType.device_meta_category ? deviceType.device_meta_category.trim() : 'N/A');
			}) as HTMLOptGroupElement | undefined;
			if (parentOptGroup) {
				parentOptGroup.appendChild(optionEl);
			} else {
				selectEl.appendChild(optionEl);
			}
		}
	} catch (error) {
		console.error("Error populating device type select:", error);
	} finally {
		selectEl.classList.remove('disabled');
		selectEl.disabled = false;
	}
}

async function initializeInventoryPage() {
	const urlParams = new URLSearchParams(window.location.search);

	for (const paramName in advSearchParams) {
		const param = advSearchParams[paramName];
		if (!param.inputElement) continue;
		initializeAdvSearchListeners([param]);
		const rawParamValue = urlParams.get(paramName);
		if (!rawParamValue) {
			param.inputElement.dataset.initialValue = "";
			if (param.negationElement) param.negationElement.checked = false;
			continue;
		}

		const decodedParam = base64ToJson(rawParamValue);
		if (decodedParam && typeof decodedParam === "object" && Object.prototype.hasOwnProperty.call(decodedParam, "param_value")) {
			param.inputElement.dataset.initialValue = decodedParam.param_value !== null && decodedParam.param_value !== undefined ? String(decodedParam.param_value) : "";
			if (param.negationElement) param.negationElement.checked = decodedParam.not === true;
			continue;
		}

		// Fallback for legacy plain query values.
		param.inputElement.dataset.initialValue = rawParamValue;
		if (param.negationElement) param.negationElement.checked = false;
	}
	if (filterManufacturer) {
		syncModelFilterAvailability();
	}

	try {
		await Promise.all([
			populateLocationSelect(advSearchParams['filter_location'].inputElement, true),
			populateBuildingRoomSelect(advSearchParams['filter_building_room'].inputElement, true),
			populateDepartmentSelect(advSearchParams['filter_department_name'].inputElement, true),
			populateManufacturerSelect(advSearchParams['filter_system_manufacturer'].inputElement, true).then(() => populateModelSelect(advSearchParams['filter_system_model'].inputElement, true)),
			populateDomainSelect(advSearchParams['filter_ad_domain'].inputElement, true),
			populateStatusSelect(advSearchParams['filter_status'].inputElement, true),
			populateDeviceTypeSelect(advSearchParams['filter_device_type'].inputElement, true)
		]);

		for (const paramName in advSearchParams) {
			const param = advSearchParams[paramName];
			if (!param.inputElement) continue;
			param.inputElement.value = param.inputElement.dataset.initialValue || "";
			handleAdvSearchInputChange([param]);
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

if (cancelUpdate) {
	cancelUpdate.addEventListener("click", (event) => {
		event.preventDefault();
		resetInventoryLookupAndUpdateForm();
		updateURLFromAdvFilters();
	});
}

const lookupTagSearchDebounceMs = 75;
let lookupTagSearchDebounceTimer: number | undefined;
let allLookupTagsCache: number[] = [];

function rebuildLookupTagCache(rawEntries?: any[]): number[] {
	const source = Array.isArray(rawEntries) ? rawEntries : window.globalLookupResults;
	allLookupTagsCache = Array.isArray(source)
		? source
			.flatMap((cache: any) => cache.entries || [])
			.map((entry: any) => entry.tagnumber)
			.filter((tag: any): tag is number => typeof tag === "number")
		: [];
	return allLookupTagsCache;
}

function renderCachedLookupTagOptions(searchTerm: string): void {
	const allTags = allLookupTagsCache.length > 0 ? allLookupTagsCache : rebuildLookupTagCache();
	const normalizedSearchTerm = (searchTerm || "").trim().toLowerCase();
	const filteredTags = normalizedSearchTerm
		? allTags.filter(tag => tag.toString().includes(normalizedSearchTerm))
		: allTags;

	if (filteredTags.includes(Number(normalizedSearchTerm))) {
		allTagsDatalist.innerHTML = '';
		return;
	}

	renderTagOptions(allTagsDatalist, filteredTags, 20);
}

if (clientLookupTagInput) {
	clientLookupTagInput.addEventListener("keyup", (event: KeyboardEvent) => {
		const searchTerm = (event.target as HTMLInputElement).value || '';
		if (lookupTagSearchDebounceTimer !== undefined) {
			clearTimeout(lookupTagSearchDebounceTimer);
		}
		lookupTagSearchDebounceTimer = window.setTimeout(() => {
			renderCachedLookupTagOptions(searchTerm);
		}, lookupTagSearchDebounceMs);
	});
}

function parseDateTimeLocalToUTC(value: string): Date | null {
	if (!value) return null;
	const parsed = new Date(value);
	return isNaN(parsed.getTime()) ? null : parsed;
}

if (updateForm) {
	updateForm.addEventListener("submit", async (event) => {
		event.preventDefault();
		submitUpdate.disabled = true;
		if (updatingInventory) return;
		updatingInventory = true;

		updateURLFromAdvFilters();

		try {
			const formObj = {} as InventoryForm;
			if (!clientLookupTagInput && !clientLookupSerial) {
				throw new Error("No tag or serial input fields found in DOM");
			}
			formObj.tagnumber = getElementNumberValue(clientLookupTagInput);
			formObj.system_serial = getElementStringValue(clientLookupSerial);
			formObj.location = getElementStringValue(locationEl);
			formObj.building = getElementStringValue(buildingUpdate);
			formObj.room = getElementStringValue(roomUpdate);
			formObj.system_manufacturer = getElementStringValue(manufacturerUpdate);
			formObj.system_model = getElementStringValue(modelUpdate);
			formObj.device_type = getElementStringValue(deviceTypeUpdate);
			formObj.department_name = getElementStringValue(departmentEl);
			formObj.ad_domain = getElementStringValue(adDomainUpdate);
			formObj.property_custodian = getElementStringValue(propertyCustodianUpdate);
			formObj.acquired_date = getElementDateValue(acquiredDateUpdate, true);
			formObj.retired_date = getElementDateValue(retiredDateUpdate, true);
			formObj.is_broken = !getElementBooleanValue(isBrokenUpdate); // invert because select options are "Is it Functional?" but backend field is "is_broken"
			formObj.disk_removed = !getElementBooleanValue(diskRemovedUpdate); // invert because select options are "Is the Disk Present?" but backend field is "disk_removed"
			formObj.last_hardware_check = lastHardwareCheckUpdate.value ? parseDateTimeLocalToUTC(lastHardwareCheckUpdate.value) : null;
			formObj.status = getElementStringValue(clientStatusUpdate);
			formObj.checkout_bool = formObj.status && statusesThatIndicateCheckout.includes(formObj.status) ? true : false;
			formObj.checkout_date = getElementDateValue(checkoutDateUpdate, true);
			formObj.return_date = getElementDateValue(returnDateUpdate, true);
			formObj.note = getElementStringValue(noteUpdate);
			// const jsonBase64 = jsonToBase64(JSON.stringify(formObj));
			// const jsonPayload = new Blob([jsonBase64], { type: "application/json" });

			const formData = new FormData();
			const jsonPayload = JSON.stringify(formObj, (_key, value) => {
				if (value instanceof Date) {
					return value.toISOString();
				}
				return value;
			});
			formData.append("json", new Blob([jsonPayload], { type: "application/json" }), "inventory.json");

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
					formData.append("inventory-update-file-input", file, fileName);
				}
			}

			const response = await fetch("/api/inventory/update", {
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
}

if (csvDownloadButton) {
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
}

if (clientMoreDetails) {
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
}

if (clientViewPhotosButton) {
	clientViewPhotosButton.addEventListener("click", (event) => {
		event.preventDefault();
		if (!clientLookupTagInput.value) return;
		const tag = clientLookupTagInput.value;
		if (tag) {
			const url = new URL(window.location.origin + '/client_images');
			url.searchParams.set('tagnumber', tag);
			window.open(url, '_blank');
		}
	});
}

if (clientAddPhotosButton) {
	clientAddPhotosButton.addEventListener("click", (event) => {
		event.preventDefault();
		fileInputUpdate.click();
	});
}

if (fileInputUpdate) {
	fileInputUpdate.addEventListener("change", () => {
		if (fileInputUpdate.files && fileInputUpdate.files.length > 0) {
			if (clientAddPhotosButton) {
				clientAddPhotosButton.textContent = `Add Photos (${fileInputUpdate.files.length})`;
				clientAddPhotosButton.classList.add("changed-input");
			}
		} else {
			if (clientAddPhotosButton) {
				clientAddPhotosButton.textContent = "Add Photos";
				clientAddPhotosButton.classList.remove("changed-input");
			}
		}
	});
}

async function uploadJSONFile(jsonFile: File): Promise<any> {
	const fileName = jsonFile.name || '';
	if (fileName.length > 100) {
		console.error(`File name ${fileName} exceeds the maximum allowed length of 100 characters`);
		alert(`File name ${fileName} exceeds the maximum allowed length of 100 characters`);
		return;
	}
	if (!allowedFileNameRegex.test(fileName)) {
		console.error(`File name ${fileName} contains invalid characters`);
		alert(`File name ${fileName} contains invalid characters`);
		return;
	}
	if (!fileName.toLowerCase().endsWith('.json')) {
		console.error(`File name ${fileName} does not have a .json extension`);
		alert(`File name ${fileName} does not have a .json extension`);
		return;
	}
	const multipartFormData = new FormData();
	multipartFormData.append("json_file", jsonFile, fileName);
	try {
		const data = await fetch(`/api/windows-client-info`, {
			method: "POST",
			body: multipartFormData
		});
		if (!data.ok) throw new Error("Server returned an error: " + data.status + " " + data.statusText);
		const jsonData = await data.json();
		if (jsonData && jsonData.tagnumber) {
			populateLocationForm(jsonData.tagnumber, undefined);
			clientLookupTagInput.value = jsonData.tagnumber.toString();
			clientLookupSerial.value = jsonData.system_serial || '';
			clientLookupWarningMessage.style.display = "block";
			clientLookupWarningMessage.textContent = "Client information populated from JSON file. Please review and submit to update inventory.";
			updateURLFromAdvFilters();
		} else {
			throw new Error("Invalid JSON data returned from server");
		}
	} catch (error) {
		console.error("Error processing JSON file:", error);
		alert("Error processing JSON file: " + (error instanceof Error ? error.message : String(error)));
		jsonFileUpload.value = "";
		jsonFileUploadButton.textContent = "Upload JSON";
		jsonFileUploadButton.classList.remove("changed-input");
	}
}


if (jsonFileUpload && jsonFileUploadButton) {
	jsonFileUploadButton.addEventListener("click", (event) => {
		event.preventDefault();
		jsonFileUpload.click();
	});
	jsonFileUpload.addEventListener("change", () => {
		if (jsonFileUpload.files && jsonFileUpload.files.length > 0) {
			const jsonFile = jsonFileUpload.files[0];
			uploadJSONFile(jsonFile);
			jsonFileUploadButton.textContent = `JSON File: ${jsonFile.name}`;
			jsonFileUploadButton.classList.add("changed-input");
		} else {
			jsonFileUploadButton.textContent = "Upload JSON";
			jsonFileUploadButton.classList.remove("changed-input");
		}
	});
}

if (clientStatusUpdate) {
	clientStatusUpdate.addEventListener("change", async () => {
		await updateCheckoutStatus();
	});
}

if (clientLookupForm) {
	clientLookupForm.addEventListener("submit", async (event) => {
		event.preventDefault();
		await submitInventoryLookup();
		for (const sec of locationFormSections) {
			if (sec && sec.length > 0) sec.forEach(s => s.style.display = "none");
		}
		for (const sec of locationFormShowSectionsButtons) {
			sec.classList.remove("selected");
			sec.blur();
		}
		if (showLocationPartButton) showLocationPartButton.classList.add("selected");
		locationPart.forEach(part => part.style.display = "flex");
		await updateCheckoutStatus();
	});
}

document.addEventListener("DOMContentLoaded", async () => {
	try {
		await initializeInventoryPage();
	} catch (e) {
		const errorMessage = e instanceof Error ? e.message : String(e);
		console.error("Error during inventory page initialization:", errorMessage);
	}
	if (Array.isArray(window.globalLookupResults) && window.globalLookupResults.length > 0) {
		const tags = rebuildLookupTagCache();
		renderTagOptions(allTagsDatalist, tags, 20);
	}

	document.addEventListener('tags:loaded', (event: Event) => {
		const customEvent = event as CustomEvent<{ entries: any[] }>;
		const rawEntries = (customEvent && customEvent.detail && Array.isArray(customEvent.detail.entries)) ? customEvent.detail.entries : window.globalLookupResults;
		const tags = rebuildLookupTagCache(rawEntries);
		renderTagOptions(allTagsDatalist, tags || [], 20);
	});
	
	locationPart.forEach(part => part.style.display = "flex");
	if (showLocationPartButton) showLocationPartButton.classList.add("selected");

	for (const button of locationFormShowSectionsButtons) {
		if (button) button.addEventListener("click", () => {
			for (const btn of locationFormShowSectionsButtons) {
				if (btn && btn !== button) {
					btn.classList.remove("selected");
				}
				btn.blur();
			}
			button.classList.add("selected");

			for (const sec of locationFormSections) {
				if (sec && sec.length > 0) sec.forEach(s => s.style.display = "none");
			}

			if (button === showLocationPartButton) {
				if (locationPart.length) locationPart.forEach(part => part.style.display = "flex");
			} else if (button === showHardwarePartButton) {
				if (hardwarePart.length) hardwarePart.forEach(part => part.style.display = "flex");
			} else if (button === showSoftwarePartButton) {
				if (softwarePart.length) softwarePart.forEach(part => part.style.display = "flex");
			} else if (button === showPropertyPartButton) {
				if (propertyPart.length) propertyPart.forEach(part => part.style.display = "flex");
			} else if (button === showNotesFilesPartButton) {
				if (notesFilesPart.length) notesFilesPart.forEach(part => part.style.display = "flex");
			}
		});
	}

	const queries = new URLSearchParams(window.location.search);
	if (queries.has("bulk_update") && queries.get("bulk_update") === "true") {
		clientLookupForm.reset();
		clientLookupForm.style.display = "none";
		bulkUpdateForm.style.display = "flex";
		if (queries.get("bulk_location")) {
			bulkUpdateLocationInput.value = queries.get("bulk_location") || '';
			bulkUpdateTagInput.focus();
		} else {
			bulkUpdateLocationInput.focus();
		}
	}

	if (toggleBulkUpdate) toggleBulkUpdate.addEventListener("click", () => {
		if (!clientLookupForm || !bulkUpdateForm) return;

		resetInventoryLookupAndUpdateForm();
		
		if (clientLookupForm.style.display !== "none") {
			clientLookupForm.reset();
			clientLookupForm.style.display = "none";
			bulkUpdateForm.style.display = "flex";
			bulkUpdateLocationInput.focus();
		} else {
			bulkUpdateForm.reset();
			bulkUpdateForm.style.display = "none";
			clientLookupForm.style.display = "flex";
			clientLookupTagInput.focus();
		}
	});
});

if (locationEl) {
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
}

if (bulkUpdateForm) {
	bulkUpdateForm.addEventListener("submit", async (event) => {
		event.preventDefault();
	});
}

if (bulkUpdateCancelButton) {
	bulkUpdateCancelButton.addEventListener("click", (event) => {
		event.preventDefault();
		bulkUpdateForm.reset();
		clientLookupTagInput.focus();
	});
}

if (bulkUpdateSubmitButton) {
	bulkUpdateSubmitButton.addEventListener("click", async (event) => {
		event.preventDefault();
		if (updatingInventory) return;
		updatingInventory = true;
		bulkUpdateSubmitButton.disabled = true;
		bulkUpdateSubmitButton.classList.add("disabled");
		const newJson: BulkUpdateRequest = {
			bulk_location: bulkUpdateLocationInput.value || null,
			bulk_tagnumbers: bulkUpdateTagInput.value ? bulkUpdateTagInput.value.split('\n').map(Number).filter(tag => !isNaN(tag) && tag > 0 && tag <= 999999) : []
		};

		if (newJson.bulk_location === null || newJson.bulk_location.trim() === "") {
			alert("Please enter a location for the bulk update.");
			updatingInventory = false;
			bulkUpdateSubmitButton.disabled = false;
			bulkUpdateSubmitButton.classList.remove("disabled");
			return;
		}
		if (newJson.bulk_tagnumbers.length === 0) {
			alert("Please enter at least one tag number for the bulk update.");
			updatingInventory = false;
			bulkUpdateSubmitButton.disabled = false;
			bulkUpdateSubmitButton.classList.remove("disabled");
			return;
		}

		for (const tag of newJson.bulk_tagnumbers) {
			if (isNaN(tag)) {
				alert(`Invalid tag number: ${tag}. Please ensure all tag numbers are valid integers.`);
				updatingInventory = false;
				bulkUpdateSubmitButton.disabled = false;
				bulkUpdateSubmitButton.classList.remove("disabled");
				return;
			}
			if (tag <= 0 || tag > 999999) {
				alert(`Tag number out of valid range (1-999999): ${tag}. Please ensure all tag numbers are within this range.`);
				updatingInventory = false;
				bulkUpdateSubmitButton.disabled = false;
				bulkUpdateSubmitButton.classList.remove("disabled");
				return;
			}
		}

		try {
			const response = await fetch("/api/inventory/bulk_update_location", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					"credentials": "include"
				},
				body: JSON.stringify(newJson)
			});
			if (!response.ok) {
				throw new Error(`Server returned an error: ${response.status} ${response.statusText}`);
			}
			const result = await response.json();
			if (result && result.updated_count !== undefined) {
				alert(`Successfully updated location for ${result.updated_count} item(s).`);
			} else {
				alert("Bulk update completed, but could not determine how many items were updated.");
			}
			await renderInventoryTable();
		} catch (error) {
			console.error("Error during bulk update:", error);
			const errorMessage = error instanceof Error ? error.message : String(error);
			alert("Error during bulk update: " + errorMessage);
		} finally {
			updatingInventory = false;
			bulkUpdateSubmitButton.disabled = false;
			bulkUpdateSubmitButton.classList.remove("disabled");
			bulkUpdateForm.reset();
		}
	});
}