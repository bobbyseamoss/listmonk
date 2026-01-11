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

// Device width configurations for preview
const deviceDimensions = {
  desktop: { width: 1200, height: 800 },
  tablet: { width: 768, height: 1024 },
  mobile: { width: 375, height: 667 },
};

interface ToolbarProps {
  onSave: () => void;
  onCancel: () => void;
}

export const Toolbar: React.FC<ToolbarProps> = ({ onSave, onCancel }) => {
  const { form, updateFormField, previewDevice, setPreviewDevice } = useFormBuilderStore();

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
            const device = deviceDimensions[previewDevice];
            const previewWindow = window.open('', '_blank', `width=${device.width},height=${device.height}`);
            if (previewWindow) {
              // Helper to get block width CSS
              const getBlockWidth = (blockType: string, width: string | undefined): string => {
                // Image blocks always take full width
                if (blockType === 'image') return '100%';
                const w = width || '100%';
                // Account for 8px gap
                const widthMap: Record<string, string> = {
                  '75%': 'calc(75% - 2px)',
                  '66%': 'calc(66.666% - 2.67px)',
                  '50%': 'calc(50% - 4px)',
                  '33%': 'calc(33.333% - 5.33px)',
                  '25%': 'calc(25% - 6px)',
                };
                return widthMap[w] || w;
              };

              // Generate block HTML with width wrappers
              const currentStep = form.steps[0]; // Preview first step
              const blocksHtml = currentStep.blocks.map(block => {
                const p = block.props as any;
                const blockWidth = getBlockWidth(block.type, p.width);
                let blockContent = '';

                switch (block.type) {
                  case 'heading':
                    blockContent = `<h${p.level} style="font-size:${p.fontSize}px;font-weight:${p.fontWeight};color:${p.color};text-align:${p.textAlign};padding:${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px;margin:0">${p.content}</h${p.level}>`;
                    break;
                  case 'text':
                    blockContent = `<div style="font-size:${p.fontSize}px;font-weight:${p.fontWeight};color:${p.color};text-align:${p.textAlign};padding:${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px">${p.content}</div>`;
                    break;
                  case 'email-input':
                    const emailLabel = p.showLabel !== false ? `<label style="display:block;margin-bottom:4px;font-size:${p.fontSize}px;color:${p.labelColor}">${p.label}${p.required ? ' <span style="color:red">*</span>' : ''}</label>` : '';
                    blockContent = `<div style="padding:${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px">${emailLabel}<input type="email" placeholder="${p.placeholder}" style="width:100%;padding:10px 12px;font-size:${p.fontSize}px;background:${p.inputBgColor};border:1px solid ${p.inputBorderColor};border-radius:${p.borderRadius}px;color:${p.inputTextColor};box-sizing:border-box"></div>`;
                    break;
                  case 'text-input':
                    blockContent = `<div style="padding:${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px"><label style="display:block;margin-bottom:4px;font-size:${p.fontSize}px;color:${p.labelColor}">${p.label}${p.required ? ' <span style="color:red">*</span>' : ''}</label><input type="text" placeholder="${p.placeholder}" style="width:100%;padding:10px 12px;font-size:${p.fontSize}px;background:${p.inputBgColor};border:1px solid ${p.inputBorderColor};border-radius:${p.borderRadius}px;color:${p.inputTextColor};box-sizing:border-box"></div>`;
                    break;
                  case 'button':
                    blockContent = `<div style="margin:${p.margin.top}px ${p.margin.right}px ${p.margin.bottom}px ${p.margin.left}px"><button style="width:${p.fullWidth ? '100%' : 'auto'};padding:${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px;font-size:${p.fontSize}px;font-weight:${p.fontWeight};color:${p.textColor};background:${p.bgColor};border:none;border-radius:${p.borderRadius}px;cursor:pointer">${p.text}</button></div>`;
                    break;
                  case 'divider':
                    blockContent = `<hr style="margin:${p.margin.top}px 0 ${p.margin.bottom}px 0;border:none;border-top:${p.thickness}px ${p.style} ${p.color}">`;
                    break;
                  case 'spacer':
                    blockContent = `<div style="height:${p.height}px"></div>`;
                    break;
                  case 'image':
                    const imgWidth = typeof p.width === 'number' ? `${p.width}px` : p.width;
                    blockContent = p.src ? `<div style="text-align:${p.alignment};padding:${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px"><img src="${p.src}" alt="${p.alt}" style="width:${imgWidth};height:${p.height};max-width:100%"></div>` : '';
                    break;
                  default:
                    blockContent = `<div style="padding:8px;background:#f3f4f6;border:1px dashed #d1d5db;border-radius:4px;font-size:12px;color:#6b7280">[${block.type}]</div>`;
                }

                // Wrap in width container
                return `<div style="width:${blockWidth};box-sizing:border-box">${blockContent}</div>`;
              }).join('');

              previewWindow.document.write(`
                <!DOCTYPE html>
                <html>
                <head>
                  <title>Form Preview - ${form.name}</title>
                  <style>
                    * { box-sizing: border-box; }
                    body {
                      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
                      margin: 0;
                      padding: 40px;
                      background: #f5f5f5;
                      min-height: 100vh;
                      display: flex;
                      align-items: center;
                      justify-content: center;
                    }
                    .form-container {
                      width: ${form.settings.width};
                      max-width: ${form.settings.maxWidth};
                      background: ${form.settings.backgroundColor};
                      border-radius: ${form.settings.borderRadius}px;
                      padding: ${form.settings.padding.top}px ${form.settings.padding.right}px ${form.settings.padding.bottom}px ${form.settings.padding.left}px;
                      box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px -1px rgba(0,0,0,0.06);
                    }
                    .blocks-container {
                      display: flex;
                      flex-wrap: wrap;
                      gap: 8px;
                      align-items: flex-end;
                    }
                    input, button { font-family: inherit; }
                    button:hover { opacity: 0.9; }
                  </style>
                </head>
                <body>
                  <div class="form-container">
                    <div class="blocks-container">
                      ${blocksHtml || '<p style="text-align:center;color:#9ca3af;width:100%">No blocks added yet</p>'}
                    </div>
                  </div>
                </body>
                </html>
              `);
              previewWindow.document.close();
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
