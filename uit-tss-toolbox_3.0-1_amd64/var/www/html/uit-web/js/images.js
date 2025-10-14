async function getImages() {
  try {
    const response = await fetch('/api/images?tagnumber=625890', {
      method: 'GET',
      headers: {
        'Content-Type': 'x-www-form-urlencoded',
      }
    });
    if (!response.ok) throw new Error("Network response was not ok");
    const returnedImage = await response.arrayBuffer();
    if (!returnedImage) throw new Error("No image data returned");
    console.log("Image data retrieved successfully");
    const blob = new Blob([returnedImage], { type: 'image/jpeg' });
    const imageUrl = URL.createObjectURL(blob);
    const imgElement = document.createElement('img');
    imgElement.src = imageUrl;
    document.getElementById('main-content').appendChild(imgElement);
  } catch (error) {
    console.error("Error reading file:", error);
  }
}