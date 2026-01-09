import React from 'react';
import {
  Box,
  Typography,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Button,
  Tooltip,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import TitleIcon from '@mui/icons-material/Title';
import TextFieldsIcon from '@mui/icons-material/TextFields';
import ImageIcon from '@mui/icons-material/Image';
import EmailIcon from '@mui/icons-material/Email';
import TextFormatIcon from '@mui/icons-material/TextFormat';
import NotesIcon from '@mui/icons-material/Notes';
import RadioButtonCheckedIcon from '@mui/icons-material/RadioButtonChecked';
import CheckBoxIcon from '@mui/icons-material/CheckBox';
import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import SmartButtonIcon from '@mui/icons-material/SmartButton';
import HorizontalRuleIcon from '@mui/icons-material/HorizontalRule';
import ExpandIcon from '@mui/icons-material/Expand';
import LocalOfferIcon from '@mui/icons-material/LocalOffer';
import TimerIcon from '@mui/icons-material/Timer';
import GroupIcon from '@mui/icons-material/Group';
import { useFormBuilderStore } from '@/store';
import { blockDefinitions } from '@/blocks/defaults';
import type { BlockType } from '@/types';

const iconMap: Record<string, React.ReactElement> = {
  title: <TitleIcon />,
  text_fields: <TextFieldsIcon />,
  image: <ImageIcon />,
  email: <EmailIcon />,
  text_format: <TextFormatIcon />,
  notes: <NotesIcon />,
  radio_button_checked: <RadioButtonCheckedIcon />,
  check_box: <CheckBoxIcon />,
  arrow_drop_down: <ArrowDropDownIcon />,
  calendar_today: <CalendarTodayIcon />,
  visibility_off: <VisibilityOffIcon />,
  smart_button: <SmartButtonIcon />,
  horizontal_rule: <HorizontalRuleIcon />,
  expand: <ExpandIcon />,
  local_offer: <LocalOfferIcon />,
  timer: <TimerIcon />,
  group: <GroupIcon />,
};

const categories = [
  { id: 'content', name: 'Content', defaultExpanded: true },
  { id: 'input', name: 'Form Inputs', defaultExpanded: true },
  { id: 'action', name: 'Actions', defaultExpanded: true },
  { id: 'layout', name: 'Layout', defaultExpanded: false },
  { id: 'special', name: 'Special', defaultExpanded: false },
];

export const BlocksSidebar: React.FC = () => {
  const { addBlock } = useFormBuilderStore();

  const handleAddBlock = (type: BlockType) => {
    addBlock(type);
  };

  return (
    <Box
      sx={{
        width: 240,
        backgroundColor: '#fff',
        borderRight: '1px solid #e5e7eb',
        overflow: 'auto',
        flexShrink: 0,
      }}
    >
      <Box sx={{ p: 2, borderBottom: '1px solid #e5e7eb' }}>
        <Typography variant="subtitle2" fontWeight="bold" color="text.secondary">
          BLOCKS
        </Typography>
      </Box>

      {categories.map((category) => {
        const blocks = blockDefinitions.filter((b) => b.category === category.id);
        if (blocks.length === 0) return null;

        return (
          <Accordion key={category.id} defaultExpanded={category.defaultExpanded} disableGutters elevation={0}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />} sx={{ minHeight: 40 }}>
              <Typography variant="body2" fontWeight="medium">
                {category.name}
              </Typography>
            </AccordionSummary>
            <AccordionDetails sx={{ p: 1, pt: 0 }}>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {blocks.map((block) => (
                  <Tooltip key={block.type} title={block.name} placement="top" arrow>
                    <Button
                      variant="outlined"
                      size="small"
                      onClick={() => handleAddBlock(block.type)}
                      sx={{
                        minWidth: 0,
                        width: 'calc(50% - 4px)',
                        p: 1,
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        gap: 0.5,
                        borderColor: '#e5e7eb',
                        color: '#374151',
                        '&:hover': {
                          borderColor: '#2563eb',
                          backgroundColor: '#eff6ff',
                        },
                      }}
                    >
                      {iconMap[block.icon] || <TextFieldsIcon />}
                      <Typography variant="caption" sx={{ fontSize: 10, lineHeight: 1.2, textAlign: 'center' }}>
                        {block.name}
                      </Typography>
                    </Button>
                  </Tooltip>
                ))}
              </Box>
            </AccordionDetails>
          </Accordion>
        );
      })}
    </Box>
  );
};
