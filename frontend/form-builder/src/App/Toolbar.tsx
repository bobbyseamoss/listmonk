import React from 'react';
import {
  AppBar,
  Toolbar as MuiToolbar,
  Typography,
  Button,
  IconButton,
  Box,
  TextField,
  FormControl,
  Select,
  MenuItem,
  Chip,
  Tooltip,
} from '@mui/material';
import SaveIcon from '@mui/icons-material/Save';
import CloseIcon from '@mui/icons-material/Close';
import VisibilityIcon from '@mui/icons-material/Visibility';
import UndoIcon from '@mui/icons-material/Undo';
import RedoIcon from '@mui/icons-material/Redo';
import PhoneIphoneIcon from '@mui/icons-material/PhoneIphone';
import DesktopWindowsIcon from '@mui/icons-material/DesktopWindows';
import TabletIcon from '@mui/icons-material/Tablet';
import { useFormBuilderStore } from '@/store';

interface ToolbarProps {
  onSave: () => void;
  onCancel: () => void;
}

export const Toolbar: React.FC<ToolbarProps> = ({ onSave, onCancel }) => {
  const { form, updateFormField } = useFormBuilderStore();
  const [previewDevice, setPreviewDevice] = React.useState<'desktop' | 'tablet' | 'mobile'>('desktop');

  const statusColors: Record<string, 'default' | 'success' | 'warning' | 'error'> = {
    draft: 'default',
    active: 'success',
    paused: 'warning',
    archived: 'error',
  };

  return (
    <AppBar position="static" color="default" elevation={1}>
      <MuiToolbar sx={{ gap: 2 }}>
        {/* Form name */}
        <TextField
          size="small"
          value={form.name}
          onChange={(e) => updateFormField('name', e.target.value)}
          placeholder="Form name"
          sx={{
            minWidth: 200,
            '& .MuiOutlinedInput-root': {
              backgroundColor: '#fff',
            },
          }}
        />

        {/* Form type indicator */}
        <FormControl size="small" sx={{ minWidth: 120 }}>
          <Select
            value={form.formType}
            onChange={(e) => updateFormField('formType', e.target.value as any)}
            sx={{ backgroundColor: '#fff' }}
          >
            <MenuItem value="popup">Popup</MenuItem>
            <MenuItem value="flyout">Flyout</MenuItem>
            <MenuItem value="fullpage">Full Page</MenuItem>
            <MenuItem value="embed">Embed</MenuItem>
            <MenuItem value="banner">Banner</MenuItem>
          </Select>
        </FormControl>

        {/* Status chip */}
        <Chip
          label={form.status.charAt(0).toUpperCase() + form.status.slice(1)}
          color={statusColors[form.status]}
          size="small"
        />

        {/* Spacer */}
        <Box sx={{ flex: 1 }} />

        {/* Preview device toggles */}
        <Box sx={{ display: 'flex', gap: 0.5, mr: 2 }}>
          <Tooltip title="Desktop Preview">
            <IconButton
              size="small"
              onClick={() => setPreviewDevice('desktop')}
              color={previewDevice === 'desktop' ? 'primary' : 'default'}
            >
              <DesktopWindowsIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title="Tablet Preview">
            <IconButton
              size="small"
              onClick={() => setPreviewDevice('tablet')}
              color={previewDevice === 'tablet' ? 'primary' : 'default'}
            >
              <TabletIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title="Mobile Preview">
            <IconButton
              size="small"
              onClick={() => setPreviewDevice('mobile')}
              color={previewDevice === 'mobile' ? 'primary' : 'default'}
            >
              <PhoneIphoneIcon />
            </IconButton>
          </Tooltip>
        </Box>

        {/* Undo/Redo (placeholder for future implementation) */}
        <Box sx={{ display: 'flex', gap: 0.5, mr: 2 }}>
          <Tooltip title="Undo">
            <span>
              <IconButton size="small" disabled>
                <UndoIcon />
              </IconButton>
            </span>
          </Tooltip>
          <Tooltip title="Redo">
            <span>
              <IconButton size="small" disabled>
                <RedoIcon />
              </IconButton>
            </span>
          </Tooltip>
        </Box>

        {/* Preview button */}
        <Button
          variant="outlined"
          startIcon={<VisibilityIcon />}
          onClick={() => {
            // Open preview in new window
            const previewWindow = window.open('', '_blank', 'width=800,height=600');
            if (previewWindow) {
              previewWindow.document.write(`
                <!DOCTYPE html>
                <html>
                <head>
                  <title>Form Preview - ${form.name}</title>
                  <style>
                    body { font-family: sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
                  </style>
                </head>
                <body>
                  <div id="form-preview">Form preview will be rendered here</div>
                </body>
                </html>
              `);
            }
          }}
        >
          Preview
        </Button>

        {/* Cancel button */}
        <Button variant="outlined" color="inherit" startIcon={<CloseIcon />} onClick={onCancel}>
          Cancel
        </Button>

        {/* Save button */}
        <Button variant="contained" color="primary" startIcon={<SaveIcon />} onClick={onSave}>
          Save
        </Button>
      </MuiToolbar>
    </AppBar>
  );
};
