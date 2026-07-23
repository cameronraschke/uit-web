async function fetchFooter(): Promise<string | null> {
  try {
    const footerContent: string | null = await fetchData("/footer", true);
    return footerContent;
  } catch (error) {
    console.error("Error fetching footer:", error);
    return null;
  }
}

function drawFooter(htmlFooterContent: string | null) {
	const footerElement = document.getElementById("uit-footer");
	if (!footerElement) {
		console.warn("Footer element not found, cannot render footer");
		return;
	}

  try {
    if (footerElement !== null && htmlFooterContent !== null) footerElement.innerHTML = htmlFooterContent;
  } catch (error) {
    console.error("Error fetching footer:", error);
  }
}

window.addEventListener("DOMContentLoaded", async () => {
	const footerContent = await fetchFooter();
	if (footerContent !== null) drawFooter(footerContent);
});