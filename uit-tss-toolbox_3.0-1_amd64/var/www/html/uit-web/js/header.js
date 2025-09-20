async function drawHeader() {
  try {
    const header = await fetchData("/header.html", true);
    if (header) document.getElementById("uit-header").innerHTML = header;
  } catch (error) {
    console.error("Error fetching header:", error);
  }
}
drawHeader();