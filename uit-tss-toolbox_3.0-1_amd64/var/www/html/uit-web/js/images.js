async function getImages(tag) {
  try {
    const response = await fetch(`/api/images?tagnumber=${tag}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'x-www-form-urlencoded',
      }
    });
    if (!response.ok) throw new Error("Network response was not ok");
    const returnedImage = await response.blob();
    if (!returnedImage) throw new Error("No image data returned");
    console.log("Image data retrieved successfully");
    const blob = new Blob([returnedImage], { type: 'image/jpeg' });
    const imageUrl = URL.createObjectURL(blob);
    const imgElement = document.createElement('img');
    imgElement.src = imageUrl;
    imgElement.style.maxWidth = '300px';
    imgElement.style.maxHeight = 'auto';
    document.getElementById('main-content').appendChild(imgElement);
    URL.revokeObjectURL(imageUrl);
  } catch (error) {
    console.error("Error reading file:", error);
  }
}