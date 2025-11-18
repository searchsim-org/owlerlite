#!/usr/bin/env node

/**
 * Creates PNG placeholder icons from base64 data
 * These are simple placeholder icons that browsers will accept
 */

const fs = require('fs');
const path = require('path');

// Base64 encoded PNG data for a simple purple square with "O" text
// These are valid minimal PNG files
const iconData = {
  // 16x16 purple square
  icon16: 'iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAACXBIWXMAAAsTAAALEwEAmpwYAAAATklEQVR4nGNgYGD4z0ABYBo1YNQAygygxgAWBgYGfgYGBhkGBoZzDAwMrxkYGP4xMDAwk2IANjMYyDSAGgMYRg0YNYAaA6jxAjU+GAIGAACRqQU6v8QV0QAAAABJRU5ErkJggg==',
  
  // 48x48 purple square
  icon48: 'iVBORw0KGgoAAAANSUhEUgAAADAAAAAwCAYAAABXAvmHAAAACXBIWXMAAAsTAAALEwEAmpwYAAAA1ElEQVR4nO2YwQ6AIAxDEf//S72oeGDBWTZcT3vYHqVbGwzDMAzDMAyj/wEReQBYC4CZewBeAF5mdgPwAHABsOZyWwCXAO4AbmbvXG4L4BrAHcDNzJ653BbALYA7gJuZPX+XWwtwsVVuLcDFVrm1ABdb5dYCXGyVWwtwsVVuLcDFVrm1ABdb5dYCXGyVWwtwsVVuLcDFVrm1ABdb5dYCXCxebAvgYvFiWwAXixfbArhYvNgWwMXixbYALhYvtgVwsXixLYCLxYttAVwsXmwL4GLxYv0HAPwBHT0wYK7fxFAAAAAASUVORK5CYII=',
  
  // 128x128 purple square
  icon128: 'iVBORw0KGgoAAAANSUhEUgAAAIAAAACACAYAAADDPmHLAAAACXBIWXMAAAsTAAALEwEAmpwYAAADM0lEQVR4nO3cv2sUQRzG8WdMYxpBRBDB/gfRQrRQECwsLLQQBC0sLLQSbGz8B2wUbCwstLGwsLEQBBsL0UKwULAQLARBEI1BjfHlDVzCcXe3N7O7M/s88IFUt8zs8+7tzO7NBAEAAAAAAABgXqnvAtbQkLRXUl/STklb20u3JL2Q9FjSvKQZSa8k/eyswrFZAzoHJZ2UdF7S/jbXblb0/9dSWP6VoP0yLelRm49A36P0Pw5JuibpnaSlmptdWfqTpLnwGJAaLb7Jg5IeRO3UdT7SXkl3O+xzJjYq38K10pDCE/Z7Tc8fSOfaff3+hNPAqPBJUvkErZB0SdKngk95Je7rvPFM2jWwStKViA/8TkPnJK0r8Xc7o/Z1eRH73G7wOdgXPvFN/z/gRtSOngdlHZR0R9JPAx/8U+FrdKz/fUnfJH0JDx2Nzgv82RdumXftq7Snx2NJ42H7bm/YvrshzR2/v/w+Y/ZXugaOStps/AOaD6dblfe5E/Z52rj+i4yuzaHx/UqvgVuGe/Fz3N6CAbVP1sBeORP79D6GPDZ+qLkkaXmNdUa98OsC68Orx6Z/mfRD0o2Kfh4Z7sVfpW03XQMvm3ggkwwPTa5U9HPbcC++ldZA35lhvBfPNu3YvP6LjK7NoXGEU5N92vIk6R9/Jf3Dr/MaWGF8cB8rdkb7YG1fB47/fwP45yAAtAAAtAAAtAAA7S8A/0VcSfQbqmhH4OWW4e8NX49eBE6F0x8BTIX06tqhrwFfDQ/GkqQjCXb2l+HvetyR9C5qv6l9bqlj//ZH7WvoP3isCB9Ck/9Ov3/lXwPndlz+DznUCEefz9I1oF8DrhlvcOiIpDMl1tmt8Nc7F/5edVvhp4eVvjvS3x40ef94UvC1T/0z0vMZvw6+W9K14F8F76vo+6/wNXR/xq+D/1G+HMxEPXFc1PG/1o1KOijpgaRvu3w2m/59bNsN9xlLQNI2ST/C8xp8cS+afjXs++VvDXI9XfjHv/2VsM0MAAAAAAAAgCBAQ/wG7hy4kaJ2q5IAAAAASUVORK5CYII='
};

// Create icons directory
const iconsDir = path.join(__dirname, 'src', 'icons');
if (!fs.existsSync(iconsDir)) {
  fs.mkdirSync(iconsDir, { recursive: true });
}

console.log('Creating PNG icons from base64 data...\n');

// Write PNG files
Object.entries(iconData).forEach(([name, data]) => {
  const filename = `${name}.png`;
  const filePath = path.join(iconsDir, filename);
  const buffer = Buffer.from(data, 'base64');
  fs.writeFileSync(filePath, buffer);
  console.log(`✓ Created ${filename}`);
});

console.log('\nPNG icons created successfully!');
console.log('These are placeholder icons. For custom icons, edit generate-icons.html or the SVG files.');

