import { Category } from './types';

/**
 * Get consistent color for a category across the application
 * First tries to use the category's color from the database,
 * then falls back to a consistent hash-based color
 */
export const getCategoryColor = (categoryName: string, categories: Category[]): string => {
  // First, try to find the category in the database and use its color
  const category = categories.find(c => c.name === categoryName);
  if (category && category.color) {
    return category.color;
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
