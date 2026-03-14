import { Category } from './types';

/**
 * Generate N visually distinct color variants from a base hex color
 * by stepping lightness and slightly shifting hue.
 */
export const generateColorVariants = (baseHex: string, count: number): string[] => {
  if (count <= 1) return [baseHex];
  const hex = baseHex.replace('#', '');
  const r = parseInt(hex.slice(0, 2), 16) / 255;
  const g = parseInt(hex.slice(2, 4), 16) / 255;
  const b = parseInt(hex.slice(4, 6), 16) / 255;
  const max = Math.max(r, g, b), min = Math.min(r, g, b), d = max - min;
  let h = 0;
  const l = (max + min) / 2;
  const s = d === 0 ? 0 : d / (l > 0.5 ? 2 - max - min : max + min);
  if (d !== 0) {
    if (max === r) h = ((g - b) / d + (g < b ? 6 : 0)) / 6;
    else if (max === g) h = ((b - r) / d + 2) / 6;
    else h = ((r - g) / d + 4) / 6;
  }
  const hslToHex = (hh: number, ss: number, ll: number): string => {
    const a = ss * Math.min(ll, 1 - ll);
    const f = (n: number) => {
      const k = (n + hh * 12) % 12;
      const val = ll - a * Math.max(-1, Math.min(k - 3, 9 - k, 1));
      return Math.round(255 * val).toString(16).padStart(2, '0');
    };
    return `#${f(0)}${f(8)}${f(4)}`;
  };
  return Array.from({ length: count }, (_, i) => {
    const t = count === 1 ? 0.5 : i / (count - 1);
    const lVariant = 0.35 + t * 0.30;
    const hShift = (h + (i - (count - 1) / 2) * 0.04 + 1) % 1;
    return hslToHex(hShift, Math.min(s + 0.1, 1), lVariant);
  });
};

/**
 * Get consistent color for a category across the application.
 * For subcategories, inherits the parent's color.
 */
export const getCategoryColor = (categoryName: string, categories: Category[]): string => {
  // Flatten the nested category tree for lookup
  const allCategories: Category[] = [];
  const flatten = (cats: Category[]) => {
    cats.forEach(c => {
      allCategories.push(c);
      if (c.subcategories) flatten(c.subcategories);
    });
  };
  flatten(categories);

  const category = allCategories.find(c => c.name === categoryName);
  if (category) {
    if (category.color) return category.color;
    // Subcategory: inherit parent's color
    if (category.parent_id) {
      const parent = allCategories.find(c => c.id === category.parent_id);
      if (parent && parent.color) return parent.color;
    }
  }

  // Fallback color palette for categories not in database or without colors
  const colorPalette = [
    '#1f77b4', '#ff7f0e', '#2ca02c', '#d62728', '#9467bd',
    '#8c564b', '#e377c2', '#7f7f7f', '#bcbd22', '#17becf',
    '#aec7e8', '#ffbb78', '#98df8a', '#ff9896', '#c5b0d5',
    '#c49c94', '#f7b6d3', '#c7c7c7', '#dbdb8d', '#9edae5'
  ];

  // Create a hash of the category name to ensure consistent color assignment
  let hash = 0;
  for (let i = 0; i < categoryName.length; i++) {
    const char = categoryName.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // Convert to 32bit integer
  }

  return colorPalette[Math.abs(hash) % colorPalette.length];
};
