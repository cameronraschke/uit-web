let updatingInventory = false;

// Inventory update form (and lookup)
const lastUpdateTimeMessage = document.getElementById('last-update-time-message') as HTMLElement;
const inventoryLookupWarningMessage = document.getElementById('existing-inventory-message') as HTMLElement;
const inventoryLookupForm = document.getElementById('inventory-lookup-form') as HTMLFormElement;
const inventoryLookupTagInput = document.getElementById('inventory-tag-lookup') as HTMLInputElement;
const inventoryLookupSystemSerialInput = document.getElementById('inventory-serial-lookup') as HTMLInputElement;
const inventoryLookupFormSubmitButton = document.getElementById('inventory-lookup-submit-button') as HTMLButtonElement;
const inventoryLookupFormResetButton = document.getElementById('inventory-lookup-reset-button') as HTMLButtonElement;
const inventoryLookupMoreDetailsButton = document.getElementById('inventory-lookup-more-details') as HTMLButtonElement;
const inventoryUpdateForm = document.getElementById('inventory-update-form') as HTMLFormElement;
const inventoryUpdateFormSection = document.getElementById('inventory-update-section') as HTMLElement;
const inventoryUpdateLocationInput = document.getElementById('location') as HTMLInputElement;
const inventoryUpdateFormSubmitButton = document.getElementById('inventory-update-submit-button') as HTMLButtonElement;
const inventoryUpdateFormCancelButton = document.getElementById('inventory-update-cancel-button') as HTMLButtonElement;
const allTagsDatalist = document.getElementById('inventory-tag-suggestions') as HTMLDataListElement;
const clientImagesLink = document.getElementById('client_images_link') as HTMLAnchorElement;
const inventoryUpdateDomainSelect = document.getElementById('ad_domain') as HTMLSelectElement;
const statusesThatIndicateBroken = ["needs-repair"];
const statusesThatIndicateCheckout = ["checked-out", "reserved-for-checkout"];

async function getTagOrSerial(tagnumber: number | null, serial: string | null): Promise<{ tagnumber: number | null; system_serial: string | null } | null> {
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
    const response: { tagnumber: number | null; system_serial: string | null } = await fetchData(`/api/lookup?${query.toString()}`);
    if (!response) {
      throw new Error("Cannot parse json from /api/lookup");
    }
    const returnObject = {
      tagnumber: response.tagnumber, 
      system_serial: response.system_serial
    };
    return returnObject;
  } catch(error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.log("Error getting tag/serial: " + errorMessage);
		return null;
  }
}

async function submitInventoryLookup() {
	await setFiltersFromURL();
	const searchParams: URLSearchParams = new URLSearchParams(window.location.search);
	const lookupTag: number | null = inventoryLookupTagInput.value ? Number(inventoryLookupTagInput.value) : (searchParams.get('tagnumber') ? Number(searchParams.get('tagnumber')) : null);
  const lookupSerial: string | null = inventoryLookupSystemSerialInput.value || searchParams.get('system_serial') || null;

  const lookupResult = await getTagOrSerial(lookupTag, lookupSerial);
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
	
	searchParams.set('update', 'true');
	searchParams.set('tagnumber', lookupTag !== null ? lookupTag.toString() : '');
	history.replaceState(null, '', window.location.pathname + '?' + searchParams.toString());
  await populateLocationForm(lookupTag !== null ? lookupTag : NaN);

  inventoryUpdateFormSection.style.display = "block";
  if (lookupResult) {
    inventoryLookupTagInput.value = lookupResult.tagnumber !== null ? lookupResult.tagnumber.toString() : "";
    inventoryLookupSystemSerialInput.value = lookupResult.system_serial || "";
    inventoryLookupTagInput.disabled = true;
    inventoryLookupSystemSerialInput.disabled = true;
    inventoryLookupTagInput.style.backgroundColor = "gainsboro";
    inventoryLookupSystemSerialInput.style.backgroundColor = "gainsboro";
    inventoryUpdateLocationInput.focus();
  } else {
    inventoryLookupWarningMessage.style.display = "block";
    inventoryLookupWarningMessage.textContent = "No inventory entry was found for the provided tag number or serial number. A new entry can be created.";
    if (!inventoryLookupTagInput.value) inventoryLookupTagInput.focus();
    else if (!inventoryLookupSystemSerialInput.value) inventoryLookupSystemSerialInput.focus();
  }

  inventoryLookupFormSubmitButton.disabled = true;
  inventoryLookupFormSubmitButton.style.cursor = "not-allowed";
  inventoryLookupFormSubmitButton.style.border = "1px solid gray";
  inventoryLookupFormResetButton.style.display = "inline-block";
  inventoryLookupMoreDetailsButton.style.display = "inline-block";
  if (lookupTag) {
    clientImagesLink.href = `/client_images?tagnumber=${lookupTag || ''}`;
    clientImagesLink.target = "_blank";
    clientImagesLink.style.display = "inline";
  } else {
    clientImagesLink.style.display = "none";
  }
}

inventoryLookupForm.addEventListener("submit", async (event) => {
  event.preventDefault();
	await submitInventoryLookup();
	await updateCheckoutStatus();
});

const clientStatus = inventoryUpdateForm.querySelector("#status") as HTMLSelectElement;
clientStatus.addEventListener("change", async () => {
	await updateCheckoutStatus();
});

async function updateCheckoutStatus() {
	const printCheckoutDiv = document.getElementById('print-checkout-link') as HTMLElement;
	if (statusesThatIndicateCheckout.includes(clientStatus.value)) {
		const printCheckoutAnchor = document.createElement('a');
		printCheckoutAnchor.setAttribute('href', `/checkout-form?tagnumber=${encodeURIComponent(inventoryLookupTagInput.value)}`);
		printCheckoutAnchor.setAttribute('target', '_blank');
		printCheckoutAnchor.textContent = 'Print Checkout Form';
		printCheckoutDiv.innerHTML = '';
		printCheckoutDiv.appendChild(printCheckoutAnchor);
	} else {
		if (printCheckoutDiv) {
			printCheckoutDiv.innerHTML = '';
		}
	}
}

function resetInventoryLookupAndUpdateForm() {
  inventoryLookupForm.reset();
  inventoryUpdateForm.reset();
  inventoryLookupTagInput.value = "";
  inventoryLookupTagInput.style.backgroundColor = "initial";
  inventoryLookupTagInput.disabled = false;
  inventoryLookupSystemSerialInput.value = "";
  inventoryLookupSystemSerialInput.style.backgroundColor = "initial";
  inventoryLookupSystemSerialInput.disabled = false;
  inventoryLookupFormSubmitButton.style.cursor = "pointer";
  inventoryLookupFormSubmitButton.style.border = "1px solid black";
  inventoryLookupFormSubmitButton.disabled = false;
  inventoryLookupFormResetButton.style.display = "none";
  inventoryLookupMoreDetailsButton.style.display = "none";
  inventoryUpdateFormSection.style.display = "none";
  inventoryLookupWarningMessage.style.display = "none";
  inventoryLookupWarningMessage.textContent = "";
	lastUpdateTimeMessage.textContent = "";
  inventoryLookupTagInput.focus();
}

inventoryLookupFormResetButton.addEventListener("click", (event) => {
  event.preventDefault();
	history.replaceState(null, '', window.location.pathname);
  resetInventoryLookupAndUpdateForm();
	updateURLParameters(null, null);
});

inventoryLookupMoreDetailsButton.addEventListener("click", (event) => {
  event.preventDefault();
  const tag = inventoryLookupTagInput.value;
  if (tag) {
    const url = `/client?tagnumber=${encodeURIComponent(tag)}`;
		window.open(url, '_blank');
  }
});

function resetInventorySearchQuery() {
	const url = new URL(window.location.pathname, window.location.origin);
	url.searchParams.delete('tagnumber');
	url.searchParams.delete('system_serial');
	url.searchParams.delete('update');
	history.replaceState(null, '', url.toString());
}

inventoryUpdateFormCancelButton.addEventListener("click", (event) => {
  event.preventDefault();
	history.replaceState(null, '', window.location.pathname);
  resetInventoryLookupAndUpdateForm();
	updateURLParameters(null, null);
});

function renderTagOptions(tags: string[]): void {
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
    option.value = tag;
    allTagsDatalist.appendChild(option);
  });
}

if (Array.isArray(window.availableTags)) {
  console.log("Available tags found:", window.availableTags);
  renderTagOptions(window.availableTags);
}

document.addEventListener('tags:loaded', (event: CustomEvent<{ tags: string[] }>) => {
  const tags = (event && event.detail && Array.isArray(event.detail.tags)) ? event.detail.tags : window.availableTags;
  renderTagOptions(tags || []);
});

inventoryLookupTagInput.addEventListener("keyup", (event: KeyboardEvent) => {
  const searchTerm = ((event.target as HTMLInputElement).value || '').trim().toLowerCase();
  const allTags = Array.isArray(window.availableTags) ? window.availableTags : [];
  const filteredTags = searchTerm
    ? allTags.filter(tag => String(tag).trim().includes(searchTerm))
    : allTags;
  if (filteredTags.includes(searchTerm)) {
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

	await setFiltersFromURL();

  try {
    const jsonObject: { [key: string]: any } = {};
    const inventoryLookupTagInput = inventoryLookupForm.querySelector("#inventory-tag-lookup") as HTMLInputElement | null;
    const inventoryLookupSystemSerialInput = inventoryLookupForm.querySelector("#inventory-serial-lookup") as HTMLInputElement | null;
    jsonObject.tagnumber = inventoryLookupTagInput && inventoryLookupTagInput.value ? Number(inventoryLookupTagInput.value) : null;
    jsonObject.system_serial = inventoryLookupSystemSerialInput && inventoryLookupSystemSerialInput.value ? String(inventoryLookupSystemSerialInput.value) : null;
    if (!inventoryLookupTagInput && !inventoryLookupSystemSerialInput) {
      throw new Error("No tag or serial input fields found in DOM");
    }
    const getInputValue = (documentID: string): string | null => {
      const input = inventoryUpdateForm.querySelector(documentID) as HTMLInputElement | null;
      return input && input.value ? String(input.value) : null;
    };
    jsonObject["location"] = getInputValue("#location");
		jsonObject["building"] = getInputValue("#building");
		jsonObject["room"] = getInputValue("#room");
    jsonObject["system_manufacturer"] = getInputValue("#system_manufacturer");
    jsonObject["system_model"] = getInputValue("#system_model");
    jsonObject["department_name"] = getInputValue("#department_name");
		jsonObject["property_custodian"] = getInputValue("#property_custodian");
    jsonObject["ad_domain"] = getInputValue("#ad_domain");
    const brokenBool = getInputValue("#is_broken");
      if (brokenBool === "true") jsonObject["is_broken"] = true;
      else if (brokenBool === "false") jsonObject["is_broken"] = false;
      else jsonObject["is_broken"] = null;
    jsonObject["status"] = getInputValue("#status");
		if (getInputValue("#acquired_date")) {
			jsonObject["acquired_date"] = new Date(getInputValue("#acquired_date") as string).toISOString() || null;
		}
    jsonObject["note"] = getInputValue("#note");

    // const jsonBase64 = jsonToBase64(JSON.stringify(jsonObject));
    // const jsonPayload = new Blob([jsonBase64], { type: "application/json" });

    const formData = new FormData();
    formData.append("json", new Blob([JSON.stringify(jsonObject)], { type: "application/json" }), "inventory.json");

    const fileInput = inventoryUpdateForm.querySelector("#inventory-file-input") as HTMLInputElement | null;
    if (fileInput && fileInput.files && fileInput.files.length > 0) {
			const fileList = Array.from(fileInput.files);
      for (const file of fileList) {
        if (!file) continue;
				let fileName: string = file.name || '';
        if (file.size > 64 * 1024 * 1024) {
          throw new Error(`File ${fileName} exceeds the maximum allowed size of 64 MB`);
        }
        if (fileName.length > 100) {
          throw new Error(`File name ${fileName} exceeds the maximum allowed length of 100 characters`);
        }
        const allowedRegex = /^[a-zA-Z0-9.\-_ ()]+\.[a-zA-Z]+$/;
        if (!allowedRegex.test(fileName)) {
          throw new Error(`File name ${fileName} contains invalid characters`);
        }
        const disallowedExtensions = [".exe", ".bat", ".sh", ".js", ".html", ".zip", ".rar", ".7z", ".tar", ".gz", ".dll", ".sys", ".ps1", ".cmd"];
        if (disallowedExtensions.some(ext => fileName.endsWith(ext))) {
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
    await populateLocationForm(returnedJson.tagnumber);
    await fetchFilteredInventoryData();
  } catch (error) {
    console.error("Error updating inventory:", error);
    const errorMessage = error instanceof Error ? error.message : String(error);
		alert("Error updating inventory: " + errorMessage);
  } finally {
    updatingInventory = false;
    inventoryUpdateFormSubmitButton.disabled = false;
  }
});

async function getLocationFormData(tag: number): Promise<any | null> {
  try {
    const response = await fetchData(`/api/client/location_form_data?tagnumber=${tag}`);
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

async function populateLocationForm(tag: number): Promise<void> {
  const locationFormData = await getLocationFormData(tag);
  if (locationFormData) {
		if (locationFormData.last_update_time) {
			const lastUpdate = new Date(locationFormData.last_update_time);
			if (isNaN(lastUpdate.getTime())) {
				lastUpdateTimeMessage.textContent = 'Uknown timestamp of last update';
			} else {
				lastUpdateTimeMessage.textContent = `Last updated: ${lastUpdate.toLocaleString()}` || '';
			}
		} else {
			lastUpdateTimeMessage.textContent = 'Uknown timestamp of last update';
		}
    inventoryUpdateLocationInput.value = locationFormData.location || '';

		await populateDomainSelect(inventoryUpdateDomainSelect);

		const building = inventoryUpdateForm.querySelector("#building") as HTMLInputElement;
		const buildingVal: string = locationFormData.building || '';
		building.value = buildingVal;

		const room = inventoryUpdateForm.querySelector("#room") as HTMLInputElement;
		const roomVal: string = locationFormData.room || '';
		room.value = roomVal;

    const systemManufacturer = inventoryUpdateForm.querySelector("#system_manufacturer") as HTMLInputElement;
		const systemManufacturerVal: string = locationFormData.system_manufacturer || '';
		systemManufacturer.value = systemManufacturerVal;

    const systemModel = inventoryUpdateForm.querySelector("#system_model") as HTMLInputElement;
		const systemModelVal: string = locationFormData.system_model || '';
		systemModel.value = systemModelVal;

    const departmentName = inventoryUpdateForm.querySelector("#department_name") as HTMLInputElement;
		const departmentNameVal: string = locationFormData.department_name || '';
		departmentName.value = departmentNameVal;

		const propertyCustodian = inventoryUpdateForm.querySelector("#property_custodian") as HTMLInputElement;
		const propertyCustodianVal: string = locationFormData.property_custodian || '';
		propertyCustodian.value = propertyCustodianVal;

    const adDomain = inventoryUpdateForm.querySelector("#ad_domain") as HTMLInputElement;
		const adDomainVal: string = locationFormData.ad_domain || '';
		adDomain.value = adDomainVal;

		const isBroken = inventoryUpdateForm.querySelector("#is_broken") as HTMLInputElement;
		const brokenValue = typeof locationFormData.is_broken === "boolean" 
			? String(locationFormData.is_broken) 
			: '';
		isBroken.value = brokenValue;

		const statusSelect = inventoryUpdateForm.querySelector("#status") as HTMLSelectElement;
    statusSelect.value = locationFormData.status || '';

		const acquiredDateInput = inventoryUpdateForm.querySelector("#acquired_date") as HTMLInputElement;
		const acquiredDateValue = locationFormData.acquired_date
			? new Date(locationFormData.acquired_date)
			: null;
		if (acquiredDateValue && !isNaN(acquiredDateValue.getTime())) {
			const year = acquiredDateValue.getFullYear();
			const month = String(acquiredDateValue.getMonth() + 1).padStart(2, '0');
			const day = String(acquiredDateValue.getDate()).padStart(2, '0');
			const acquiredDateFormatted = `${year}-${month}-${day}`;
			acquiredDateInput.value = acquiredDateFormatted;
		} else {
			acquiredDateInput.value = '';
		}

    const noteInput = inventoryUpdateForm.querySelector("#note") as HTMLInputElement;
    const noteValue: string = locationFormData.note || '';
    noteInput.value = noteValue;
  }
	await updateCheckoutStatus();
}

const csvDownloadButton = document.getElementById('inventory-search-download-button') as HTMLButtonElement;
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

async function initializeInventoryPage() {
	await loadAllManufacturersAndModels();
	await setFiltersFromURL();
	await initializeSearch();
	await populateModelSelect(filterManufacturer.value || null);
	await populateDomainSelect(filterDomain);
  await fetchFilteredInventoryData();
	const urlParams = new URLSearchParams(window.location.search);
	const updateParam: string | null = urlParams.get('update');
	const tagnumberParam: string | null = urlParams.get('tagnumber');
	if (tagnumberParam && updateParam === 'true') {
		inventoryLookupTagInput.value = tagnumberParam;
		await submitInventoryLookup();
	}
}

document.addEventListener("DOMContentLoaded", async () => {
  await initializeInventoryPage();
});

