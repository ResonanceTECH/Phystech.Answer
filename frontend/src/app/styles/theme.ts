import { createTheme } from '@mui/material/styles'

// Дизайн-токены Физтех.Ответ — единый визуальный язык с лендингом:
// «violet-noir», тёмная основа + фиолетовый акцент, Unbounded/Onest.
const VIOLET = '#8b5cf6'
const VIOLET_BRIGHT = '#a78bfa'
const VIOLET_DEEP = '#6d28d9'
const BORDER = 'rgba(167, 139, 250, 0.16)'
const BORDER_STRONG = 'rgba(167, 139, 250, 0.32)'

const fontBody = "'Onest', 'Inter', system-ui, 'Segoe UI', sans-serif"
const fontDisplay = "'Unbounded', 'Onest', system-ui, sans-serif"

export const appTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: VIOLET,
      light: VIOLET_BRIGHT,
      dark: VIOLET_DEEP,
      contrastText: '#ffffff',
    },
    secondary: {
      main: '#a89fc4',
    },
    background: {
      default: '#0a0713',
      paper: '#151024',
    },
    text: {
      primary: '#f5f3fc',
      secondary: '#a89fc4',
      disabled: '#6e658c',
    },
    divider: BORDER,
    success: { main: '#34d399' },
    warning: { main: '#fbbf24' },
    error: { main: '#f87171' },
    info: { main: VIOLET_BRIGHT },
    action: {
      hover: 'rgba(167, 139, 250, 0.08)',
      selected: 'rgba(139, 92, 246, 0.22)',
    },
  },
  typography: {
    fontFamily: fontBody,
    h4: {
      fontFamily: fontDisplay,
      fontWeight: 700,
      letterSpacing: '-0.02em',
      lineHeight: 1.08,
      textTransform: 'uppercase',
    },
    h5: {
      fontFamily: fontDisplay,
      fontWeight: 700,
      letterSpacing: '-0.02em',
      lineHeight: 1.15,
    },
    h6: {
      fontFamily: fontDisplay,
      fontWeight: 700,
      letterSpacing: '-0.01em',
      lineHeight: 1.2,
    },
    body1: { lineHeight: 1.6 },
    body2: { lineHeight: 1.55 },
    button: { fontWeight: 600 },
    caption: { color: '#6e658c' },
  },
  shape: {
    borderRadius: 14,
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: { colorScheme: 'dark' },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          borderRadius: 999,
          paddingInline: '1.25rem',
          transition:
            'transform 0.25s ease, background-color 0.25s ease, border-color 0.25s ease, box-shadow 0.25s ease',
          '&:hover': { transform: 'translateY(-1px)' },
          '&:active': { transform: 'translateY(0)' },
        },
        containedPrimary: {
          boxShadow: '0 14px 40px rgba(124, 58, 237, 0.32)',
          '&:hover': { backgroundColor: VIOLET_BRIGHT },
        },
        outlinedPrimary: {
          background: '#1d1335',
          borderColor: BORDER_STRONG,
          color: VIOLET_BRIGHT,
          '&:hover': {
            background: '#241840',
            borderColor: VIOLET,
            color: '#f5f3fc',
          },
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: { backgroundImage: 'none' },
        rounded: { borderRadius: 20 },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 20,
          backgroundColor: '#151024',
          borderColor: BORDER,
          transition:
            'transform 0.25s ease, border-color 0.25s ease, box-shadow 0.25s ease',
        },
      },
    },
    MuiCardActionArea: {
      styleOverrides: {
        root: {
          '&:hover': { transform: 'translateY(-2px)' },
          transition: 'transform 0.25s ease',
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundColor: 'rgba(10, 7, 19, 0.82)',
          backdropFilter: 'blur(18px)',
          WebkitBackdropFilter: 'blur(18px)',
          backgroundImage: 'none',
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          backgroundColor: '#0f0a1c',
          backgroundImage: 'none',
        },
      },
    },
    MuiListItemButton: {
      styleOverrides: {
        root: {
          borderRadius: 999,
          '&.active': {
            backgroundColor: 'rgba(139, 92, 246, 0.22)',
            color: VIOLET_BRIGHT,
            '& .MuiListItemIcon-root': { color: VIOLET_BRIGHT },
          },
          '&:hover': { backgroundColor: 'rgba(167, 139, 250, 0.08)' },
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: { fontWeight: 500 },
        outlined: {
          backgroundColor: '#1d1335',
          borderColor: BORDER_STRONG,
          color: VIOLET_BRIGHT,
          '&:hover': { backgroundColor: '#241840' },
        },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          backgroundColor: '#1d1335',
          '& fieldset': { borderColor: BORDER },
          '&:hover fieldset': { borderColor: BORDER_STRONG },
          '&.Mui-focused fieldset': { borderColor: VIOLET },
        },
      },
    },
    MuiDialog: {
      styleOverrides: {
        paper: { backgroundColor: '#151024', border: `1px solid ${BORDER}` },
      },
    },
    MuiTooltip: {
      styleOverrides: {
        tooltip: { backgroundColor: '#2a1656', fontSize: '0.8rem' },
      },
    },
    MuiDivider: {
      styleOverrides: {
        root: { borderColor: BORDER },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: { borderColor: BORDER },
      },
    },
  },
})
