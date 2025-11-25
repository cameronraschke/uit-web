let updatingInventory = false;

// Inventory update form (and lookup)
const lastUpdateTimeMessage = document.getElementById('last-update-time-message');
const inventoryLookupWarningMessage = document.getElementById('existing-inventory-message');
const inventoryLookupForm = document.getElementById('inventory-lookup-form');
const inventoryLookupTagInput = document.getElementById('inventory-tag-lookup');
const inventoryLookupSystemSerialInput = document.getElementById('inventory-serial-lookup');
const inventoryLookupFormSubmitButton = document.getElementById('inventory-lookup-submit-button');
const inventoryLookupFormResetButton = document.getElementById('inventory-lookup-reset-button');
const clientMoreDetailsButton = document.getElementById('client-more-details');
const inventoryUpdateForm = document.getElementById('inventory-update-form');
const inventoryUpdateFormSection = document.getElementById('inventory-update-section');
const inventoryUpdateLocationInput = document.getElementById('location');
const inventoryUpdateFormSubmitButton = document.getElementById('inventory-update-submit-button');
const inventoryUpdateFormCancelButton = document.getElementById('inventory-update-cancel-button');
const allTagsDatalist = document.getElementById('inventory-tag-suggestions');
const clientImagesLink = document.getElementById('client_images_link');

const statusesThatIndicateBroken = ["needs-repair"];
const statusesThatIndicateCheckout = ["checked-out", "reserved-for-checkout"];

async function getTagOrSerial(tagnumber, serial) {
  const query = new URLSearchParams();
  if (tagnumber) {
    query.append("tagnumber", tagnumber);
  } else if (serial) {
    query.append("system_serial", serial);
  } else {
    console.log("No tag or serial provided");
    return;
  }
  try {
    const response = await fetchData(`/api/lookup?${query.toString()}`);
    if (!response) {
      throw new Error("Cannot parse json from /api/lookup");
    }
    const returnObject = {
      tagnumber: response.tagnumber, 
      system_serial: response.system_serial
    };
    return returnObject;
  } catch(error) {
    console.log("Error getting tag/serial: " + error.message);
  }
}

async function submitInventoryLookup() {
	await setFiltersFromURL();
	const searchParams = new URLSearchParams(window.location.search);
	const lookupTag = inventoryLookupTagInput.value || searchParams.get('tagnumber') || null;
  const lookupSerial = inventoryLookupSystemSerialInput.value || searchParams.get('system_serial') || null;

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
  if (lookupTag && lookupTag.length != 6) {
    inventoryLookupWarningMessage.style.display = "block";
    inventoryLookupWarningMessage.textContent = "Tag number must be exactly 6 digits long.";
    return;
  }
	
	history.replaceState(null, '', window.location.pathname + `?update=true&tagnumber=${encodeURIComponent(lookupTag || '')}`);
  await populateLocationForm(Number(lookupTag));

  inventoryUpdateFormSection.style.display = "block";
  if (lookupResult) {
    inventoryLookupTagInput.value = lookupResult.tagnumber || "";
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
  clientMoreDetailsButton.style.display = "inline-block";
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
	await updateBrokenStatus();
});

const clientStatus = inventoryUpdateForm.querySelector("#status");
clientStatus.addEventListener("change", async () => {
	await updateCheckoutStatus();
	await updateBrokenStatus();
});

async function updateCheckoutStatus() {
	const printCheckoutDiv = document.getElementById('print-checkout-link');
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

async function updateBrokenStatus() {
	if (statusesThatIndicateBroken.includes(clientStatus.value)) {
		inventoryUpdateForm.querySelector("#is_broken").value = "true";
	} else {
		inventoryUpdateForm.querySelector("#is_broken").value = "false";
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
  clientMoreDetailsButton.style.display = "none";
  inventoryUpdateFormSection.style.display = "none";
  inventoryLookupWarningMessage.style.display = "none";
  inventoryLookupWarningMessage.textContent = "";
	lastUpdateTimeMessage.textContent = "";
  inventoryLookupTagInput.focus();
}

inventoryLookupFormResetButton.addEventListener("click", (event) => {
  event.preventDefault();
  resetInventoryLookupAndUpdateForm();
	updateURLParameters();
});

clientMoreDetailsButton.addEventListener("click", (event) => {
  event.preventDefault();
  const tag = inventoryLookupTagInput.value;
  if (tag) {
    window.location.href = `/client/${tag}`;
  }
});

function resetInventorySearchQuery() {
	url = new URL(window.location);
	url.searchParams.delete('tagnumber');
	url.searchParams.delete('system_serial');
	url.searchParams.delete('update');
	history.replaceState(null, '', url.toString());
}

inventoryUpdateFormCancelButton.addEventListener("click", (event) => {
  event.preventDefault();
  resetInventoryLookupAndUpdateForm();
	updateURLParameters();
});

function renderTagOptions(tags) {
  if (!allTagsDatalist) {
    console.warn("No tag datalist found");
    return;
  }
  
  allTagsDatalist.innerHTML = '';
  let maxTags = 20;
  if (checkMobile()) {
    maxTags = 0;
    return;
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

document.addEventListener('tags:loaded', (event) => {
  const tags = (event && event.detail && Array.isArray(event.detail.tags)) ? event.detail.tags : window.availableTags;
  renderTagOptions(tags || []);
});

inventoryLookupTagInput.addEventListener("keyup", (event) => {
  const searchTerm = (event.target.value || '').trim().toLowerCase();
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
    const jsonObject = {};
    const inventoryLookupTagInput = inventoryLookupForm.querySelector("#inventory-tag-lookup");
    const inventoryLookupSystemSerialInput = inventoryLookupForm.querySelector("#inventory-serial-lookup");
    jsonObject.tagnumber = inventoryLookupTagInput && inventoryLookupTagInput.value ? Number(inventoryLookupTagInput.value) : null;
    jsonObject.system_serial = inventoryLookupSystemSerialInput && inventoryLookupSystemSerialInput.value ? String(inventoryLookupSystemSerialInput.value) : null;
    if (!inventoryLookupTagInput && !inventoryLookupSystemSerialInput) {
      throw new Error("No tag or serial input fields found in DOM");
    }
    const getInputValue = (documentID) => {
      const input = inventoryUpdateForm.querySelector(documentID);
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
			jsonObject["acquired_date"] = new Date(getInputValue("#acquired_date")).toISOString() || null;
		}
    jsonObject["note"] = getInputValue("#note");

    // const jsonBase64 = jsonToBase64(JSON.stringify(jsonObject));
    // const jsonPayload = new Blob([jsonBase64], { type: "application/json" });

    const formData = new FormData();
    formData.append("json", new Blob([JSON.stringify(jsonObject)], { type: "application/json" }), "inventory.json");

    const fileInput = inventoryUpdateForm.querySelector("#inventory-file-input");
    if (fileInput && fileInput.files && fileInput.files.length > 0) {
      for (const file of fileInput.files) {
        if (!file) continue;
        if (file.size > 64 * 1024 * 1024) {
          throw new Error(`File ${file.name} exceeds the maximum allowed size of 64 MB`);
        }
        if (file.name.length > 100) {
          throw new Error(`File name ${file.name} exceeds the maximum allowed length of 100 characters`);
        }
        const allowedRegex = /^[a-zA-Z0-9.\-_ ()]+\.[a-zA-Z]+$/;
        if (!allowedRegex.test(file.name)) {
          throw new Error(`File name ${file.name} contains invalid characters`);
        }
        const disallowedExtensions = [".exe", ".bat", ".sh", ".js", ".html", ".zip", ".rar", ".7z", ".tar", ".gz", ".dll", ".sys", ".ps1", ".cmd"];
        if (disallowedExtensions.some(ext => file.name.endsWith(ext))) {
          throw new Error(`File name ${file.name} has a forbidden extension`);
        }
        if (file.name.endsWith(".jfif")) {
          file.name = file.name.replace(/\.jfif$/i, ".jpeg");
        }
        formData.append("inventory-file-input", file, file.name);
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
		fileInput.value = "";
		inventoryLookupWarningMessage.style.display = "none";
		lastUpdateTimeMessage.textContent = "";
    await populateLocationForm(Number(returnedJson.tagnumber));
    await fetchFilteredInventoryData();
  } catch (error) {
    console.error("Error updating inventory:", error);
		alert("Error updating inventory: " + error.message);
  } finally {
    updatingInventory = false;
    inventoryUpdateFormSubmitButton.disabled = false;
  }
});

async function getLocationFormData(tag) {
  try {
    const response = await fetchData(`/api/client/location_form_data?tagnumber=${tag}`);
    if (!response) {
      throw new Error("Cannot parse json from /api/client/location_form_data");
    }
    return response;
  } catch (error) {
    console.log("Error fetching location form data: " + error.message);
    return null;
  }
}

async function populateLocationForm(tag) {
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
		inventoryUpdateForm.querySelector("#building").value = locationFormData.building || '';
		inventoryUpdateForm.querySelector("#room").value = locationFormData.room || '';
    inventoryUpdateForm.querySelector("#system_manufacturer").value = locationFormData.system_manufacturer || '';
    inventoryUpdateForm.querySelector("#system_model").value = locationFormData.system_model || '';
    inventoryUpdateForm.querySelector("#department_name").value = locationFormData.department_name || '';
		inventoryUpdateForm.querySelector("#property_custodian").value = locationFormData.property_custodian || '';
    inventoryUpdateForm.querySelector("#ad_domain").value = locationFormData.ad_domain || '';
		const brokenValue = typeof locationFormData.is_broken === "boolean" 
			? String(locationFormData.is_broken) 
			: '';
		inventoryUpdateForm.querySelector("#is_broken").value = brokenValue;
    inventoryUpdateForm.querySelector("#status").value = locationFormData.status || '';
		const acquiredDateValue = locationFormData.acquired_date
			? new Date(locationFormData.acquired_date)
			: null;
		const parsedDate = acquiredDateValue.toISOString() || null;
		if (!isNaN(parsedDate) && parsedDate && parsedDate instanceof Date && parsedDate > 0) {
			const acquiredDate = new Date(parsedDate);
			const year = acquiredDate.getFullYear();
			const month = String(acquiredDate.getMonth() + 1).padStart(2, '0');
			const day = String(acquiredDate.getDate()).padStart(2, '0');
			const acquiredDateFormatted = `${year}-${month}-${day}`;
			acquiredDateValue = acquiredDateFormatted || '';
		}
    inventoryUpdateForm.querySelector("#note").value = locationFormData.note || '';
  }
	await updateCheckoutStatus();
	await updateBrokenStatus();
}

const csvDownloadButton = document.getElementById('inventory-search-download-button');
csvDownloadButton.addEventListener('click', async (event) => {
  event.preventDefault();
  csvDownloadButton.disabled = true;
  csvDownloadButton.textContent = 'Preparing download...';
  try {
    await fetchFilteredInventoryData(true);
  } finally {
    initializeInventoryPage();
    csvDownloadButton.disabled = false;
    csvDownloadButton.textContent = 'Download Results';
  }
});

async function initializeInventoryPage() {
	await loadAllManufacturersAndModels();
	await setFiltersFromURL();
	await initializeSearch();
	await populateModelSelect(filterManufacturer.value || null);
  await fetchFilteredInventoryData()
	const urlParams = new URLSearchParams(window.location.search);
	if (urlParams.get('tagnumber') && urlParams.get('update') === 'true') {
		inventoryLookupTagInput.value = urlParams.get('tagnumber');
		await submitInventoryLookup();
	}
}

document.addEventListener("DOMContentLoaded", async () => {
  await initializeInventoryPage();
});

