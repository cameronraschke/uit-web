let updatingInventory = false;

const inventoryLookupWarningMessage = document.getElementById('existing-inventory-message');
const inventoryLookupForm = document.getElementById('inventory-lookup-form');
const inventoryLookupTagInput = document.getElementById('inventory-tag-lookup');
const inventoryLookupSerialInput = document.getElementById('inventory-serial-lookup');
const inventoryLookupSubmitButton = document.getElementById('inventory-lookup-submit-button');
const inventoryLookupResetButton = document.getElementById('inventory-lookup-reset-button');
const inventoryUpdateForm = document.getElementById('inventory-update-form');
const inventoryUpdateSection = document.getElementById('inventory-update-section');
const inventoryLocationInput = document.getElementById('location');
const inventoryUpdateSubmitButton = document.getElementById('inventory-update-submit-button');
const inventoryUpdateCancelButton = document.getElementById('inventory-update-cancel-button');
const tagDatalist = document.getElementById('inventory-tag-suggestions');

function postInventoryData() {
  return null;
}

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
  inventoryUpdateSection.style.display = "none";
  inventoryLookupWarningMessage.style.display = "none";
  inventoryLookupWarningMessage.textContent = "";
  inventoryLookupTagInput.focus();
}

inventoryLookupResetButton.addEventListener("click", (event) => {
  event.preventDefault();
  resetInventoryForm();
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
  (tags || []).slice(0, 20).forEach(tag => {
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

  const jsonObject = {};
  if (inventoryLookupTagInput && inventoryLookupSerialInput) {
    const lookupTag = inventoryLookupForm.querySelector("#inventory-tag-lookup").value;
    lookupTag ? jsonObject["tagnumber"] = Number(lookupTag) : jsonObject["tagnumber"] = null;
    inventoryLookupForm.querySelector("#inventory-serial-lookup").value ? jsonObject["system_serial"] = String(inventoryLookupForm.querySelector("#inventory-serial-lookup").value) : jsonObject["system_serial"] = null;
    inventoryUpdateForm.querySelector("#location").value ? jsonObject["location"] = String(inventoryUpdateForm.querySelector("#location").value) : jsonObject["location"] = null;
    inventoryUpdateForm.querySelector("#system_manufacturer").value ? jsonObject["system_manufacturer"] = String(inventoryUpdateForm.querySelector("#system_manufacturer").value) : jsonObject["system_manufacturer"] = null;
    inventoryUpdateForm.querySelector("#system_model").value ? jsonObject["system_model"] = String(inventoryUpdateForm.querySelector("#system_model").value) : jsonObject["system_model"] = null;
    inventoryUpdateForm.querySelector("#department").value ? jsonObject["department"] = String(inventoryUpdateForm.querySelector("#department").value) : jsonObject["department"] = null;
    inventoryUpdateForm.querySelector("#domain").value ? jsonObject["domain"] = String(inventoryUpdateForm.querySelector("#domain").value) : jsonObject["domain"] = null;
    inventoryUpdateForm.querySelector("#working").value ? jsonObject["working"] = new Boolean(inventoryUpdateForm.querySelector("#working").value) : jsonObject["working"] = null;
    inventoryUpdateForm.querySelector("#status").value ? jsonObject["status"] = String(inventoryUpdateForm.querySelector("#status").value) : jsonObject["status"] = null;
    inventoryUpdateForm.querySelector("#note").value ? jsonObject["note"] = String(inventoryUpdateForm.querySelector("#note").value) : jsonObject["note"] = null;
    inventoryUpdateForm.querySelector("#inventory-file-input").files.length > 0 ? jsonObject["image"] = "" : jsonObject["image"] = null;

    var fileCount = 0;
    for (const file of inventoryUpdateForm.querySelector("#inventory-file-input").files) {
      fileCount++;
      formData.append("image-" + fileCount, file);
    }
  } else {
    throw new Error("No tag or serial input fields found in DOM");
  }

  const jsonPayload = jsonToBase64(JSON.stringify(jsonObject));

  try {
    const response = await fetch("/api/update_inventory", {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: jsonPayload
    });

    const data = await response.json();

    if (!response.ok) {
      throw new Error("Failed to update inventory");
    }
    const returnedJson = JSON.parse(base64ToJson(data));
    if (!returnedJson) {
      throw new Error("No return data from inventory update");
    }
    await populateLocationForm(lookupTag);
    console.log("Inventory updated successfully");
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
    if (typeof locationFormData.working === "boolean") inventoryUpdateForm.querySelector("#working").value = locationFormData.working;
    if (typeof locationFormData.status === "boolean") inventoryUpdateForm.querySelector("#status").value = locationFormData.status;
    if (locationFormData.note) inventoryUpdateForm.querySelector("#note").value = locationFormData.note;
  }
}