#!/usr/bin/env node

/**
 * Simple icon generator for OwlerLite extension
 * Creates basic PNG icons without external dependencies
 */

const fs = require('fs');
const path = require('path');

// Create icons directory if it doesn't exist
const iconsDir = path.join(__dirname, 'src', 'icons');
if (!fs.existsSync(iconsDir)) {
  fs.mkdirSync(iconsDir, { recursive: true });
}

// SVG template for the icon
function createSVG(size) {
  return `<?xml version="1.0" encoding="UTF-8"?>
<svg width="${size}" height="${size}" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <linearGradient id="grad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#667eea;stop-opacity:1" />
      <stop offset="100%" style="stop-color:#764ba2;stop-opacity:1" />
    </linearGradient>
  </defs>
  
  <!-- Background -->
  <rect width="${size}" height="${size}" fill="url(#grad)" />
  
  <!-- Owl body -->
  <ellipse cx="${size/2}" cy="${size*0.55}" rx="${size*0.3}" ry="${size*0.35}" fill="white" />
  
  <!-- Eyes -->
  <circle cx="${size*0.4}" cy="${size*0.45}" r="${size*0.08}" fill="#667eea" />
  <circle cx="${size*0.6}" cy="${size*0.45}" r="${size*0.08}" fill="#667eea" />
  
  <!-- Pupils -->
  <circle cx="${size*0.4}" cy="${size*0.45}" r="${size*0.04}" fill="white" />
  <circle cx="${size*0.6}" cy="${size*0.45}" r="${size*0.04}" fill="white" />
  
  <!-- Beak -->
  <polygon points="${size*0.5},${size*0.52} ${size*0.46},${size*0.58} ${size*0.54},${size*0.58}" fill="#fbbf24" />
  
  <!-- Wings -->
  <ellipse cx="${size*0.32}" cy="${size*0.6}" rx="${size*0.08}" ry="${size*0.12}" fill="#e5e7eb" transform="rotate(-20 ${size*0.32} ${size*0.6})" />
  <ellipse cx="${size*0.68}" cy="${size*0.6}" rx="${size*0.08}" ry="${size*0.12}" fill="#e5e7eb" transform="rotate(20 ${size*0.68} ${size*0.6})" />
</svg>`;
}

// Write SVG files
const sizes = [16, 48, 128];

console.log('Generating OwlerLite icons...\n');

sizes.forEach(size => {
  const svgContent = createSVG(size);
  const svgPath = path.join(iconsDir, `icon${size}.svg`);
  
  fs.writeFileSync(svgPath, svgContent);
  console.log(`✓ Created icon${size}.svg`);
});

console.log('\nSVG icons created successfully!');
console.log('\nTo convert SVG to PNG:');
console.log('1. Open generate-icons.html in your browser');
console.log('2. Download PNG files');
console.log('3. Place them in src/icons/');
console.log('\nOr use ImageMagick:');
console.log('  convert src/icons/icon16.svg src/icons/icon16.png');
console.log('  convert src/icons/icon48.svg src/icons/icon48.png');
console.log('  convert src/icons/icon128.svg src/icons/icon128.png');

