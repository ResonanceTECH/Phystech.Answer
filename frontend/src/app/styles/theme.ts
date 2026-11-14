import { createTheme } from '@mui/material/styles'

// Дизайн-токены Физтех.Ответ: спокойный интерфейс с акцентом
// на читаемости и работе с документами.
export const appTheme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#1d4e9e',
      dark: '#163d7d',
      light: '#4a73c2',
      contrastText: '#ffffff',
    },
    secondary: {
      main: '#475569',
    },
    background: {
      default: '#f6f7f9',
      paper: '#ffffff',
    },
    text: {
      primary: '#1f2430',
      secondary: '#5b6472',
    },
    success: { main: '#2e7d4f' },
    warning: { main: '#b45309' },
    error: { main: '#c62828' },
    info: { main: '#1d4e9e' },
  },
  typography: {
    fontFamily:
      "'Inter', system-ui, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
    h4: { fontWeight: 650, letterSpacing: '-0.01em' },
    h5: { fontWeight: 650, letterSpacing: '-0.01em' },
    h6: { fontWeight: 600 },
    body1: { lineHeight: 1.6 },
    body2: { lineHeight: 1.55 },
  },
  shape: {
    borderRadius: 10,
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: { textTransform: 'none', fontWeight: 550 },
      },
    },
    MuiPaper: {
      styleOverrides: {
        rounded: { borderRadius: 14 },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: { borderRadius: 14 },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: { fontWeight: 500 },
      },
    },
    MuiListItemButton: {
      styleOverrides: {
        root: { borderRadius: 8 },
      },
    },
  },
})
