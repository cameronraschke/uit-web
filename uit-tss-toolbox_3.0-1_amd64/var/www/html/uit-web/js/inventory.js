let updatingInventory = false;

// Inventory update form (and lookup)
const inventoryLookupWarningMessage = document.getElementById('existing-inventory-message');
const inventoryLookupForm = document.getElementById('inventory-lookup-form');
const inventoryLookupTagInput = document.getElementById('inventory-tag-lookup');
const inventoryLookupSerialInput = document.getElementById('inventory-serial-lookup');
const inventoryLookupSubmitButton = document.getElementById('inventory-lookup-submit-button');
const inventoryLookupResetButton = document.getElementById('inventory-lookup-reset-button');
const clientMoreDetailsButton = document.getElementById('client-more-details');
const inventoryUpdateForm = document.getElementById('inventory-update-form');
const inventoryUpdateSection = document.getElementById('inventory-update-section');
const inventoryLocationInput = document.getElementById('location');
const inventoryUpdateSubmitButton = document.getElementById('inventory-update-submit-button');
const inventoryUpdateCancelButton = document.getElementById('inventory-update-cancel-button');
const tagDatalist = document.getElementById('inventory-tag-suggestions');
const clientImagesLink = document.getElementById('client_images_link');

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
    const request = await fetchData(`/api/lookup?${query.toString()}`);
    if (!request) {
      throw new Error("Cannot parse json from /api/lookup");
    }

    const returnObject = {
      tagnumber: request.tagnumber, 
      system_serial: request.system_serial
    }

    return returnObject;

  } catch(error) {
    console.log("Error getting tag/serial: " + error.message);
  }
}

inventoryLookupForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const lookupTag = inventoryLookupTagInput.value;
  const lookupSerial = inventoryLookupSerialInput.value;
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
  await populateLocationForm(lookupTag);

  inventoryUpdateSection.style.display = "block";
  if (lookupResult) {
    inventoryLookupTagInput.value = lookupResult.tagnumber || "";
    inventoryLookupSerialInput.value = lookupResult.system_serial || "";
    inventoryLookupTagInput.disabled = true;
    inventoryLookupSerialInput.disabled = true;
    inventoryLookupTagInput.style.backgroundColor = "gainsboro";
    inventoryLookupSerialInput.style.backgroundColor = "gainsboro";
    inventoryLocationInput.focus();
  } else {
    inventoryLookupWarningMessage.style.display = "block";
    inventoryLookupWarningMessage.textContent = "No inventory entry was found for the provided tag number or serial number. A new entry can be created.";
    if (!inventoryLookupTagInput.value) inventoryLookupTagInput.focus();
    else if (!inventoryLookupSerialInput.value) inventoryLookupSerialInput.focus();
  }

  inventoryLookupSubmitButton.disabled = true;
  inventoryLookupSubmitButton.style.cursor = "not-allowed";
  inventoryLookupSubmitButton.style.border = "1px solid gray";
  inventoryLookupResetButton.style.display = "inline-block";
  clientMoreDetailsButton.style.display = "inline-block";
  if (lookupTag) {
    clientImagesLink.href = `/client_images?tagnumber=${lookupTag || ''}`;
    clientImagesLink.target = "_blank";
    clientImagesLink.style.display = "inline";
  } else {
    clientImagesLink.style.display = "none";
  }
});

function resetInventoryForm() {
  inventoryLookupForm.reset();
  inventoryUpdateForm.reset();
  inventoryLookupTagInput.value = "";
  inventoryLookupTagInput.style.backgroundColor = "initial";
  inventoryLookupTagInput.disabled = false;
  inventoryLookupSerialInput.value = "";
  inventoryLookupSerialInput.style.backgroundColor = "initial";
  inventoryLookupSerialInput.disabled = false;
  inventoryLookupSubmitButton.style.cursor = "pointer";
  inventoryLookupSubmitButton.style.border = "1px solid black";
  inventoryLookupSubmitButton.disabled = false;
  inventoryLookupResetButton.style.display = "none";
  clientMoreDetailsButton.style.display = "none";
  inventoryUpdateSection.style.display = "none";
  inventoryLookupWarningMessage.style.display = "none";
  inventoryLookupWarningMessage.textContent = "";
  inventoryLookupTagInput.focus();
}

inventoryLookupResetButton.addEventListener("click", (event) => {
  event.preventDefault();
  resetInventoryForm();
});

clientMoreDetailsButton.addEventListener("click", (event) => {
  event.preventDefault();
  const tag = inventoryLookupTagInput.value;
  if (tag) {
    window.location.href = `/client/${tag}`;
  }
});

inventoryUpdateCancelButton.addEventListener("click", (event) => {
  event.preventDefault();
  resetInventoryForm();
});

function renderTagOptions(tags) {
  if (!tagDatalist) {
    console.warn("No tag datalist found");
    return;
  }
  
  tagDatalist.innerHTML = '';
  let maxTags = 20;
  if (checkMobile()) {
    maxTags = 0;
    return;
  }
  (tags || []).slice(0, maxTags).forEach(tag => {
    const option = document.createElement('option');
    option.value = tag;
    tagDatalist.appendChild(option);
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
    tagDatalist.innerHTML = '';
  } else {
    renderTagOptions(filteredTags);
  }
});

inventoryUpdateForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  inventoryUpdateSubmitButton.disabled = true;
  if (updatingInventory) return;
  updatingInventory = true;

  try {
    const jsonObject = {};
    const inventoryLookupTagInput = inventoryLookupForm.querySelector("#inventory-tag-lookup");
    const inventoryLookupSerialInput = inventoryLookupForm.querySelector("#inventory-serial-lookup");
    jsonObject.tagnumber = inventoryLookupTagInput && inventoryLookupTagInput.value ? Number(inventoryLookupTagInput.value) : null;
    jsonObject.system_serial = inventoryLookupSerialInput && inventoryLookupSerialInput.value ? String(inventoryLookupSerialInput.value) : null;
    if (!inventoryLookupTagInput && !inventoryLookupSerialInput) {
      throw new Error("No tag or serial input fields found in DOM");
    }
    const getInputValue = (documentID) => {
      const input = inventoryUpdateForm.querySelector(documentID);
      return input && input.value ? String(input.value) : null;
    };
    jsonObject["location"] = getInputValue("#location");
    jsonObject["system_manufacturer"] = getInputValue("#system_manufacturer");
    jsonObject["system_model"] = getInputValue("#system_model");
    jsonObject["department"] = getInputValue("#department");
    jsonObject["domain"] = getInputValue("#domain");
    const brokenBool = getInputValue("#broken");
      if (brokenBool === "true") jsonObject["broken"] = true;
      else if (brokenBool === "false") jsonObject["broken"] = false;
      else jsonObject["broken"] = null;
    jsonObject["status"] = getInputValue("#status");
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
      throw new Error("Failed to update inventory");
    }

    const data = await response.json();
    if (!data || !data.json) {
      throw new Error("No return data from inventory update");
    }
    const returnedJson = JSON.parse(base64ToJson(data.json));
    if (!returnedJson) {
      throw new Error("No return data from inventory update");
    }
    await populateLocationForm(Number(returnedJson.tagnumber));
    renderInventoryTable();
  } catch (error) {
    console.error("Error updating inventory:", error);
  } finally {
    updatingInventory = false;
    inventoryUpdateSubmitButton.disabled = false;
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
  if (locationFormData && location) {
    if (locationFormData.location) inventoryLocationInput.value = locationFormData.location;
    if (locationFormData.system_manufacturer) inventoryUpdateForm.querySelector("#system_manufacturer").value = locationFormData.system_manufacturer;
    if (locationFormData.system_model) inventoryUpdateForm.querySelector("#system_model").value = locationFormData.system_model;
    if (locationFormData.department) inventoryUpdateForm.querySelector("#department").value = locationFormData.department;
    if (locationFormData.domain) inventoryUpdateForm.querySelector("#domain").value = locationFormData.domain;
    if (typeof locationFormData.is_broken === "boolean") inventoryUpdateForm.querySelector("#broken").value = locationFormData.is_broken;
    if (typeof locationFormData.status === "string") inventoryUpdateForm.querySelector("#status").value = locationFormData.status;
    if (locationFormData.note) inventoryUpdateForm.querySelector("#note").value = locationFormData.note;
  }
}

const csvDownloadButton = document.getElementById('inventory-filter-download-button');
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

function initializeInventoryPage() {
  Promise.all([fetchFilteredInventoryData(), loadManufacturersAndModels()]);
}
document.addEventListener("DOMContentLoaded", () => {
  initializeInventoryPage();
});

