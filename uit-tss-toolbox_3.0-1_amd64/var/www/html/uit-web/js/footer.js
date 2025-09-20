async function drawFooter() {
  try {
    const footer = await fetchData("/footer.html", true);
    if (footer) document.getElementById("uit-footer").innerHTML = footer;
  } catch (error) {
    console.error("Error fetching footer:", error);
  }
}
drawFooter();