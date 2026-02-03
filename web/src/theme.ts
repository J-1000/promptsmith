// PromptSmith Theme - Terminal-inspired dark aesthetic
// Forge metaphor: amber glow, dark steel, craftsmanship

export const colors = {
  // Background layers (dark forge)
  bg: {
    primary: '#0a0a0f',
    secondary: '#12121a',
    tertiary: '#1a1a24',
    elevated: '#22222e',
  },

  // Text
  text: {
    primary: '#e8e6e3',
    secondary: '#9a9a9a',
    muted: '#5a5a6a',
    inverse: '#0a0a0f',
  },

  // Accent - Amber/Gold (forge fire)
  accent: {
    primary: '#f5a623',
    secondary: '#d4910f',
    muted: '#8a6b2f',
    glow: 'rgba(245, 166, 35, 0.15)',
  },

  // Semantic
  success: '#4ade80',
  warning: '#fbbf24',
  error: '#ef4444',
  info: '#60a5fa',

  // Diff colors
  diff: {
    added: '#22543d',
    addedText: '#86efac',
    removed: '#7f1d1d',
    removedText: '#fca5a5',
    context: '#1a1a24',
  },

  // Borders
  border: {
    subtle: '#2a2a3a',
    default: '#3a3a4a',
    focus: '#f5a623',
  },
} as const

export const fonts = {
  mono: "'JetBrains Mono', 'Fira Code', 'SF Mono', Menlo, monospace",
  display: "'Space Grotesk', system-ui, sans-serif",
} as const

export const spacing = {
  xs: '4px',
  sm: '8px',
  md: '16px',
  lg: '24px',
  xl: '32px',
  xxl: '48px',
} as const

export const radius = {
  sm: '4px',
  md: '8px',
  lg: '12px',
} as const
