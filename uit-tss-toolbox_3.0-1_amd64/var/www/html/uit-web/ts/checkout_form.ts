type CheckoutData = {
	tagnumber: number | null;
	customer_name: string | null;
	customer_psid: string | null;
	checkout_date: string | null;
	return_date: string | null;
};

async function fetchCheckoutData(): Promise<CheckoutData> {
	const urlParams = new URLSearchParams(window.location.search);
	const tagNumber = urlParams.get("tagnumber");
	if (!tagNumber) {
		throw new Error("Tag number is required");
	}
	const response = await fetch(`/api/checkout?tagnumber=${encodeURIComponent(tagNumber)}`);
	if (!response.ok) {
		throw new Error(`Failed to fetch checkout data: ${response.statusText}`);
	}
	const data: CheckoutData = await response.json();
	return data;
}

document.addEventListener('DOMContentLoaded', async () => {
	const printCheckoutFormContainer = document.getElementById('print-checkout-form-container');
	const customerNameEl = document.getElementById('print-customer-name') as HTMLInputElement | null;
	const customerPSIDEl = document.getElementById('print-customer-psid') as HTMLInputElement | null;
	const checkoutDateEl = document.getElementById('print-checkout-date') as HTMLInputElement | null;
	const returnDateEl = document.getElementById('print-return-date') as HTMLInputElement | null;
	const confirmAndPrint = document.getElementById('confirm-and-print') as HTMLElement | null;

	const checkoutData = await fetchCheckoutData();
	if (customerNameEl) customerNameEl.value = checkoutData.customer_name ?? '';
	if (customerPSIDEl) customerPSIDEl.value = checkoutData.customer_psid ?? '';
	if (checkoutData.checkout_date !== null) {
		const checkoutDate = checkoutData.checkout_date ?? null;
		const checkoutDateFormatted: string | undefined = new Date(checkoutDate).toISOString().split('T')[0];
		if (checkoutDateEl && checkoutDateEl.value !== undefined) {
			checkoutDateEl.value = checkoutDateFormatted ?? '';
		}
	}
	if (checkoutData.return_date !== null) {
		const returnDate = checkoutData.return_date ?? null;
		const returnDateFormatted: string | undefined = new Date(returnDate).toISOString().split('T')[0];
		if (returnDateEl && returnDateEl.value !== undefined) {
			returnDateEl.value = returnDateFormatted ?? '';
		}
	}
	
	const options: Intl.DateTimeFormatOptions = {
		timeZone: "America/Chicago",
		year: "numeric",
		month: "long",
		weekday: "long",
		day: "numeric",
	};

	confirmAndPrint?.addEventListener('click', () => {

		const customerName = customerNameEl?.value.trim() ?? '';
		const customerPSID = customerPSIDEl?.value.trim() ?? '';
		const checkoutDateStr = checkoutDateEl?.value.trim() ?? '';
		const returnDateStr = returnDateEl?.value.trim() ?? '';


		(document.querySelector('#customer_name') as HTMLElement).textContent = customerName;
		(document.querySelector('#customer_psid') as HTMLElement).textContent = customerPSID;
		(document.querySelector('#checkout_date') as HTMLElement).textContent =  checkoutDateStr ? new Date(checkoutDateStr + "T00:00:00").toLocaleDateString('en-US', options) : '';
		(document.querySelector('#return_date') as HTMLElement).textContent = returnDateStr ? new Date(returnDateStr + "T00:00:00").toLocaleDateString('en-US', options) : '';

		if (customerName || customerPSID) {
			const urlParams = new URLSearchParams(window.location.search)
			let fileName = urlParams.get("tagnumber") + " Checkout Form";
			if (customerName) fileName = fileName + ` - ${customerName}`;
			if (customerPSID) fileName = fileName + ` (${customerPSID})`;
			window.document.title = fileName;
		}

		window.print();

		window.addEventListener('afterprint', () => {
			printCheckoutFormContainer!.style.display = 'block';
    }, { once: true });
	});
});