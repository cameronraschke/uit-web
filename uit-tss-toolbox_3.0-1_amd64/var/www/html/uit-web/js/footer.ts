async function drawFooter() {
	const footerElement = document.getElementById("uit-footer");
	if (!footerElement) return;

  try {
    const footerContent = await fetchData("/footer", true);
    if (footerElement) footerElement.innerHTML = footerContent;
  } catch (error) {
    console.error("Error fetching footer:", error);
  }
}
drawFooter();