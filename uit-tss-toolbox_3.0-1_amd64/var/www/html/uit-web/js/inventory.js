const inventoryLookupButton = document.getElementById('inventory-lookup-button');

function postInventoryData() {
  return null;
}

async function getTagOrSerial() {
  try {
    const request = fetchData('/api/lookup');
    if (!request) {
      throw new Error("Cannot parse json from /api/lookup");
    }
    const tagnumber = request.tagnumber
    const serial = request.system_serial

    console.log(tagnumber + serial);

  } catch(e) {
    console.log("Error getting tag/serial:" + e.error());
  }
}

inventoryLookupButton.addEventListener("click", async () => {
  await getTagOrSerial();
});