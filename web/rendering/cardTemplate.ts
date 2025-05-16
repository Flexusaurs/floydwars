export interface CardTemplateConfig {
  base: string;
  overlays?: string[];
  width: number;
  height: number;
}

export async function renderCard(config: CardTemplateConfig): Promise<HTMLCanvasElement> {
  const canvas = document.createElement('canvas');
  canvas.width = config.width;
  canvas.height = config.height;
  const ctx = canvas.getContext('2d')!;
  ctx.imageSmoothingEnabled = false;

  const loadImage = (src: string): Promise<HTMLImageElement> =>
    new Promise((resolve, reject) => {
      const img = new Image();
      img.onload = () => resolve(img);
      img.onerror = reject;
      img.src = src;
    });

  const baseImg = await loadImage(config.base);
  const overlayImgs = config.overlays
    ? await Promise.all(config.overlays.map(loadImage))
    : [];

  ctx.drawImage(baseImg, 0, 0, config.width, config.height);
  for (const img of overlayImgs) {
    ctx.drawImage(img, 0, 0, config.width, config.height);
  }

  return canvas;
}
