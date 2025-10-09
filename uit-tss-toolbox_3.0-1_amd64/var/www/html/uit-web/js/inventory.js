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

  } catch(e) {
    console.log("Error getting tag/serial: " + e.message);
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
  inventoryLookupTagInput.value = "";
  inventoryLookupSerialInput.value = "";
  inventoryLookupTagInput.disabled = false;
  inventoryLookupSerialInput.disabled = false;
  inventoryLookupSubmitButton.disabled = false;
  inventoryLookupSubmitButton.style.cursor = "pointer";
  inventoryLookupSubmitButton.style.border = "1px solid black";
  inventoryLookupResetButton.style.display = "none";
  inventoryLookupForm.reset();
  inventoryUpdateForm.reset();
  inventoryUpdateSection.style.display = "none";
  inventoryLookupWarningMessage.style.display = "none";
  inventoryLookupWarningMessage.textContent = "";
  inventoryLookupTagInput.style.backgroundColor = "initial";
  inventoryLookupSerialInput.style.backgroundColor = "initial";
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
