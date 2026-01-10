import React from 'react';
import {
  Box,
  Typography,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Switch,
  FormControlLabel,
  Slider,
  Divider,
  Tabs,
  Tab,
  IconButton,
  Accordion,
  AccordionSummary,
  AccordionDetails,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { HexColorPicker } from 'react-colorful';
import { useFormBuilderStore } from '@/store';
import { blockDefinitions } from '@/blocks/defaults';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div role="tabpanel" hidden={value !== index} {...other}>
      {value === index && <Box sx={{ p: 2 }}>{children}</Box>}
    </div>
  );
}

const ColorInput: React.FC<{ label: string; value: string; onChange: (color: string) => void }> = ({
  label,
  value,
  onChange,
}) => {
  const [showPicker, setShowPicker] = React.useState(false);

  return (
    <Box sx={{ mb: 2 }}>
      <Typography variant="caption" color="text.secondary" gutterBottom>
        {label}
      </Typography>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Box
          onClick={() => setShowPicker(!showPicker)}
          sx={{
            width: 32,
            height: 32,
            backgroundColor: value,
            border: '1px solid #d1d5db',
            borderRadius: 1,
            cursor: 'pointer',
          }}
        />
        <TextField
          size="small"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          sx={{ flex: 1 }}
        />
      </Box>
      {showPicker && (
        <Box sx={{ mt: 1 }}>
          <HexColorPicker color={value} onChange={onChange} />
        </Box>
      )}
    </Box>
  );
};

const PaddingInput: React.FC<{
  value: { top: number; right: number; bottom: number; left: number };
  onChange: (padding: { top: number; right: number; bottom: number; left: number }) => void;
}> = ({ value, onChange }) => (
  <Box sx={{ mb: 2 }}>
    <Typography variant="caption" color="text.secondary" gutterBottom>
      Padding
    </Typography>
    <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 1 }}>
      <TextField
        size="small"
        label="Top"
        type="number"
        value={value.top}
        onChange={(e) => onChange({ ...value, top: parseInt(e.target.value) || 0 })}
      />
      <TextField
        size="small"
        label="Right"
        type="number"
        value={value.right}
        onChange={(e) => onChange({ ...value, right: parseInt(e.target.value) || 0 })}
      />
      <TextField
        size="small"
        label="Bottom"
        type="number"
        value={value.bottom}
        onChange={(e) => onChange({ ...value, bottom: parseInt(e.target.value) || 0 })}
      />
      <TextField
        size="small"
        label="Left"
        type="number"
        value={value.left}
        onChange={(e) => onChange({ ...value, left: parseInt(e.target.value) || 0 })}
      />
    </Box>
  </Box>
);

const BlockPropertiesPanel: React.FC = () => {
  const { form, selectedBlockId, selectedStepIndex, updateBlock } = useFormBuilderStore();
  const currentStep = form.steps[selectedStepIndex];
  const selectedBlock = currentStep.blocks.find((b) => b.id === selectedBlockId);

  if (!selectedBlock) {
    return (
      <Box sx={{ p: 3, textAlign: 'center', color: '#9ca3af' }}>
        <Typography variant="body2">Select a block to edit its properties</Typography>
      </Box>
    );
  }

  const blockDef = blockDefinitions.find((d) => d.type === selectedBlock.type);
  const props = selectedBlock.props as Record<string, any>;

  const handleChange = (key: string, value: any) => {
    updateBlock(selectedBlock.id, { [key]: value });
  };

  return (
    <Box sx={{ p: 2 }}>
      <Typography variant="subtitle2" fontWeight="bold" gutterBottom>
        {blockDef?.name || selectedBlock.type}
      </Typography>
      <Divider sx={{ mb: 2 }} />

      {/* Common text properties */}
      {props.content !== undefined && (
        <TextField
          fullWidth
          multiline
          rows={3}
          size="small"
          label="Content"
          value={props.content}
          onChange={(e) => handleChange('content', e.target.value)}
          sx={{ mb: 2 }}
        />
      )}

      {props.text !== undefined && (
        <TextField
          fullWidth
          size="small"
          label="Text"
          value={props.text}
          onChange={(e) => handleChange('text', e.target.value)}
          sx={{ mb: 2 }}
        />
      )}

      {props.label !== undefined && (
        <TextField
          fullWidth
          size="small"
          label="Label"
          value={props.label}
          onChange={(e) => handleChange('label', e.target.value)}
          sx={{ mb: 2 }}
        />
      )}

      {props.placeholder !== undefined && (
        <TextField
          fullWidth
          size="small"
          label="Placeholder"
          value={props.placeholder}
          onChange={(e) => handleChange('placeholder', e.target.value)}
          sx={{ mb: 2 }}
        />
      )}

      {props.name !== undefined && (
        <TextField
          fullWidth
          size="small"
          label="Field Name"
          value={props.name}
          onChange={(e) => handleChange('name', e.target.value)}
          sx={{ mb: 2 }}
        />
      )}

      {props.required !== undefined && (
        <FormControlLabel
          control={
            <Switch
              checked={props.required}
              onChange={(e) => handleChange('required', e.target.checked)}
            />
          }
          label="Required"
          sx={{ mb: 2 }}
        />
      )}

      {props.fontSize !== undefined && (
        <Box sx={{ mb: 2 }}>
          <Typography variant="caption" color="text.secondary">
            Font Size: {props.fontSize}px
          </Typography>
          <Slider
            value={props.fontSize}
            onChange={(_, value) => handleChange('fontSize', value)}
            min={10}
            max={48}
            valueLabelDisplay="auto"
          />
        </Box>
      )}

      {props.level !== undefined && (
        <FormControl fullWidth size="small" sx={{ mb: 2 }}>
          <InputLabel>Heading Level</InputLabel>
          <Select
            value={props.level}
            label="Heading Level"
            onChange={(e) => handleChange('level', e.target.value)}
          >
            {[1, 2, 3, 4, 5, 6].map((level) => (
              <MenuItem key={level} value={level}>
                H{level}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      )}

      {props.textAlign !== undefined && (
        <FormControl fullWidth size="small" sx={{ mb: 2 }}>
          <InputLabel>Text Align</InputLabel>
          <Select
            value={props.textAlign}
            label="Text Align"
            onChange={(e) => handleChange('textAlign', e.target.value)}
          >
            <MenuItem value="left">Left</MenuItem>
            <MenuItem value="center">Center</MenuItem>
            <MenuItem value="right">Right</MenuItem>
          </Select>
        </FormControl>
      )}

      {props.action !== undefined && (
        <FormControl fullWidth size="small" sx={{ mb: 2 }}>
          <InputLabel>Action</InputLabel>
          <Select
            value={props.action}
            label="Action"
            onChange={(e) => handleChange('action', e.target.value)}
          >
            <MenuItem value="submit">Submit Form</MenuItem>
            <MenuItem value="next-step">Next Step</MenuItem>
            <MenuItem value="close">Close Form</MenuItem>
            <MenuItem value="redirect">Redirect</MenuItem>
          </Select>
        </FormControl>
      )}

      {props.fullWidth !== undefined && (
        <FormControlLabel
          control={
            <Switch
              checked={props.fullWidth}
              onChange={(e) => handleChange('fullWidth', e.target.checked)}
            />
          }
          label="Full Width"
          sx={{ mb: 2 }}
        />
      )}

      {props.width !== undefined && (
        <FormControl fullWidth size="small" sx={{ mb: 2 }}>
          <InputLabel>Block Width</InputLabel>
          <Select
            value={props.width}
            label="Block Width"
            onChange={(e) => handleChange('width', e.target.value)}
          >
            <MenuItem value="100%">Full Width (100%)</MenuItem>
            <MenuItem value="50%">Half Width (50%)</MenuItem>
            <MenuItem value="33%">Third Width (33%)</MenuItem>
            <MenuItem value="25%">Quarter Width (25%)</MenuItem>
          </Select>
        </FormControl>
      )}

      {props.borderRadius !== undefined && (
        <Box sx={{ mb: 2 }}>
          <Typography variant="caption" color="text.secondary">
            Border Radius: {props.borderRadius}px
          </Typography>
          <Slider
            value={props.borderRadius}
            onChange={(_, value) => handleChange('borderRadius', value)}
            min={0}
            max={20}
            valueLabelDisplay="auto"
          />
        </Box>
      )}

      {/* Colors */}
      {props.color !== undefined && (
        <ColorInput label="Text Color" value={props.color} onChange={(v) => handleChange('color', v)} />
      )}
      {props.textColor !== undefined && (
        <ColorInput label="Text Color" value={props.textColor} onChange={(v) => handleChange('textColor', v)} />
      )}
      {props.bgColor !== undefined && (
        <ColorInput label="Background Color" value={props.bgColor} onChange={(v) => handleChange('bgColor', v)} />
      )}
      {props.labelColor !== undefined && (
        <ColorInput label="Label Color" value={props.labelColor} onChange={(v) => handleChange('labelColor', v)} />
      )}
      {props.inputBgColor !== undefined && (
        <ColorInput label="Input Background" value={props.inputBgColor} onChange={(v) => handleChange('inputBgColor', v)} />
      )}
      {props.inputBorderColor !== undefined && (
        <ColorInput label="Input Border" value={props.inputBorderColor} onChange={(v) => handleChange('inputBorderColor', v)} />
      )}

      {/* Padding */}
      {props.padding !== undefined && (
        <PaddingInput value={props.padding} onChange={(v) => handleChange('padding', v)} />
      )}

      {/* Image specific */}
      {props.src !== undefined && (
        <TextField
          fullWidth
          size="small"
          label="Image URL"
          value={props.src}
          onChange={(e) => handleChange('src', e.target.value)}
          sx={{ mb: 2 }}
        />
      )}
      {props.alt !== undefined && (
        <TextField
          fullWidth
          size="small"
          label="Alt Text"
          value={props.alt}
          onChange={(e) => handleChange('alt', e.target.value)}
          sx={{ mb: 2 }}
        />
      )}

      {/* Spacer */}
      {props.height !== undefined && selectedBlock.type === 'spacer' && (
        <Box sx={{ mb: 2 }}>
          <Typography variant="caption" color="text.secondary">
            Height: {props.height}px
          </Typography>
          <Slider
            value={props.height}
            onChange={(_, value) => handleChange('height', value)}
            min={8}
            max={120}
            valueLabelDisplay="auto"
          />
        </Box>
      )}
    </Box>
  );
};

const FormSettingsPanel: React.FC = () => {
  const { form, updateSettings, updateFormField } = useFormBuilderStore();
  const settings = form.settings;

  return (
    <Box sx={{ p: 2 }}>
      <Accordion defaultExpanded>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2">General</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <TextField
            fullWidth
            size="small"
            label="Form Name"
            value={form.name}
            onChange={(e) => updateFormField('name', e.target.value)}
            sx={{ mb: 2 }}
          />
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>Form Type</InputLabel>
            <Select
              value={form.formType}
              label="Form Type"
              onChange={(e) => updateFormField('formType', e.target.value as any)}
            >
              <MenuItem value="popup">Popup</MenuItem>
              <MenuItem value="flyout">Flyout</MenuItem>
              <MenuItem value="fullpage">Full Page</MenuItem>
              <MenuItem value="embed">Embed</MenuItem>
              <MenuItem value="banner">Banner</MenuItem>
            </Select>
          </FormControl>
        </AccordionDetails>
      </Accordion>

      <Accordion>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2">Appearance</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <TextField
            fullWidth
            size="small"
            label="Width"
            value={settings.width}
            onChange={(e) => updateSettings({ width: e.target.value })}
            sx={{ mb: 2 }}
          />
          <ColorInput
            label="Background Color"
            value={settings.backgroundColor}
            onChange={(v) => updateSettings({ backgroundColor: v })}
          />
          <Box sx={{ mb: 2 }}>
            <Typography variant="caption" color="text.secondary">
              Border Radius: {settings.borderRadius}px
            </Typography>
            <Slider
              value={settings.borderRadius}
              onChange={(_, value) => updateSettings({ borderRadius: value as number })}
              min={0}
              max={32}
            />
          </Box>
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>Animation</InputLabel>
            <Select
              value={settings.animation}
              label="Animation"
              onChange={(e) => updateSettings({ animation: e.target.value as any })}
            >
              <MenuItem value="none">None</MenuItem>
              <MenuItem value="fade">Fade</MenuItem>
              <MenuItem value="slide">Slide</MenuItem>
              <MenuItem value="zoom">Zoom</MenuItem>
            </Select>
          </FormControl>
        </AccordionDetails>
      </Accordion>

      <Accordion>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2">Overlay</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <FormControlLabel
            control={
              <Switch
                checked={settings.overlay}
                onChange={(e) => updateSettings({ overlay: e.target.checked })}
              />
            }
            label="Show Overlay"
            sx={{ mb: 2 }}
          />
          {settings.overlay && (
            <>
              <ColorInput
                label="Overlay Color"
                value={settings.overlayColor}
                onChange={(v) => updateSettings({ overlayColor: v })}
              />
              <Box sx={{ mb: 2 }}>
                <Typography variant="caption" color="text.secondary">
                  Overlay Opacity: {Math.round(settings.overlayOpacity * 100)}%
                </Typography>
                <Slider
                  value={settings.overlayOpacity}
                  onChange={(_, value) => updateSettings({ overlayOpacity: value as number })}
                  min={0}
                  max={1}
                  step={0.1}
                />
              </Box>
              <FormControlLabel
                control={
                  <Switch
                    checked={settings.closeOnOverlayClick}
                    onChange={(e) => updateSettings({ closeOnOverlayClick: e.target.checked })}
                  />
                }
                label="Close on Overlay Click"
              />
            </>
          )}
        </AccordionDetails>
      </Accordion>

      <Accordion>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="subtitle2">Success Action</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>On Success</InputLabel>
            <Select
              value={settings.successAction}
              label="On Success"
              onChange={(e) => updateSettings({ successAction: e.target.value as any })}
            >
              <MenuItem value="message">Show Message</MenuItem>
              <MenuItem value="redirect">Redirect</MenuItem>
              <MenuItem value="close">Close Form</MenuItem>
            </Select>
          </FormControl>
          {settings.successAction === 'message' && (
            <TextField
              fullWidth
              multiline
              rows={2}
              size="small"
              label="Success Message"
              value={settings.successMessage}
              onChange={(e) => updateSettings({ successMessage: e.target.value })}
            />
          )}
          {settings.successAction === 'redirect' && (
            <TextField
              fullWidth
              size="small"
              label="Redirect URL"
              value={settings.successRedirectUrl}
              onChange={(e) => updateSettings({ successRedirectUrl: e.target.value })}
            />
          )}
        </AccordionDetails>
      </Accordion>
    </Box>
  );
};

export const PropertiesPanel: React.FC = () => {
  const { selectedBlockId } = useFormBuilderStore();
  const [tabIndex, setTabIndex] = React.useState(0);

  // Switch to block tab when a block is selected
  React.useEffect(() => {
    if (selectedBlockId) {
      setTabIndex(0);
    }
  }, [selectedBlockId]);

  return (
    <Box
      sx={{
        width: 300,
        backgroundColor: '#fff',
        borderLeft: '1px solid #e5e7eb',
        overflow: 'auto',
        flexShrink: 0,
      }}
    >
      <Tabs value={tabIndex} onChange={(_, v) => setTabIndex(v)} variant="fullWidth">
        <Tab label="Block" />
        <Tab label="Form" />
      </Tabs>
      <TabPanel value={tabIndex} index={0}>
        <BlockPropertiesPanel />
      </TabPanel>
      <TabPanel value={tabIndex} index={1}>
        <FormSettingsPanel />
      </TabPanel>
    </Box>
  );
};
