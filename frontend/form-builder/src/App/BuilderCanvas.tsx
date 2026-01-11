import React from 'react';
import { Box, Typography, IconButton, Tabs, Tab } from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import AddIcon from '@mui/icons-material/Add';
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { useFormBuilderStore } from '@/store';
import { BlockRenderer } from '@/blocks/BlockRenderer';
import type { Block } from '@/types';

// Helper to get block width from props - accounts for gap between blocks
const getBlockWidth = (block: Block): string => {
  // Image blocks always take full width - image alignment is handled inside the block
  if (block.type === 'image') {
    return '100%';
  }

  const props = block.props as Record<string, unknown>;
  const width = (props.width as string) || '100%';

  // When blocks are less than 100%, account for the 8px gap (gap: 1 = 8px in MUI)
  // The formula is: calc(X% - gap * (1 - X/100)) where gap = 8px
  const widthMap: Record<string, string> = {
    '75%': 'calc(75% - 2px)',
    '66%': 'calc(66.666% - 2.67px)',
    '50%': 'calc(50% - 4px)',
    '33%': 'calc(33.333% - 5.33px)',
    '25%': 'calc(25% - 6px)',
  };

  return widthMap[width] || width;
};

interface SortableBlockProps {
  block: Block;
  isSelected: boolean;
  onSelect: () => void;
  onDelete: () => void;
  onDuplicate: () => void;
}

const SortableBlock: React.FC<SortableBlockProps> = ({
  block,
  isSelected,
  onSelect,
  onDelete,
  onDuplicate,
}) => {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: block.id,
  });

  const blockWidth = getBlockWidth(block);

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <Box
      ref={setNodeRef}
      style={style}
      sx={{
        position: 'relative',
        backgroundColor: '#fff',
        borderRadius: 1,
        border: isSelected ? '2px solid #2563eb' : '1px solid transparent',
        cursor: 'pointer',
        width: blockWidth,
        // Account for gap when block is less than 100% width
        flexShrink: 0,
        boxSizing: 'border-box',
        '&:hover': {
          border: isSelected ? '2px solid #2563eb' : '1px solid #d1d5db',
        },
        '&:hover .block-actions': {
          opacity: 1,
        },
      }}
      onClick={(e) => {
        e.stopPropagation();
        onSelect();
      }}
    >
      {/* Drag handle and actions */}
      <Box
        className="block-actions"
        sx={{
          position: 'absolute',
          top: -1,
          right: -1,
          display: 'flex',
          gap: 0.5,
          backgroundColor: '#fff',
          borderRadius: '0 4px 0 4px',
          boxShadow: 1,
          opacity: isSelected ? 1 : 0,
          transition: 'opacity 0.2s',
          zIndex: 10,
        }}
      >
        <IconButton
          size="small"
          {...attributes}
          {...listeners}
          sx={{ cursor: 'grab', '&:active': { cursor: 'grabbing' } }}
        >
          <DragIndicatorIcon fontSize="small" />
        </IconButton>
        <IconButton size="small" onClick={(e) => { e.stopPropagation(); onDuplicate(); }}>
          <ContentCopyIcon fontSize="small" />
        </IconButton>
        <IconButton size="small" onClick={(e) => { e.stopPropagation(); onDelete(); }} color="error">
          <DeleteIcon fontSize="small" />
        </IconButton>
      </Box>

      {/* Block content */}
      <BlockRenderer block={block} />
    </Box>
  );
};

// Device width configurations
const deviceWidths = {
  desktop: { maxWidth: '100%', label: 'Desktop' },
  tablet: { maxWidth: '768px', label: 'Tablet' },
  mobile: { maxWidth: '375px', label: 'Mobile' },
};

export const BuilderCanvas: React.FC = () => {
  const {
    form,
    selectedBlockId,
    selectedStepIndex,
    selectBlock,
    removeBlock,
    duplicateBlock,
    selectStep,
    addStep,
    removeStep,
    updateStepName,
    previewDevice,
  } = useFormBuilderStore();

  const currentStep = form.steps[selectedStepIndex];
  const deviceConfig = deviceWidths[previewDevice];

  const handleTabChange = (_: React.SyntheticEvent, newValue: number) => {
    if (newValue === form.steps.length) {
      addStep();
    } else {
      selectStep(newValue);
    }
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
      {/* Step tabs */}
      {form.steps.length > 1 && (
        <Box sx={{ mb: 2, backgroundColor: '#fff', borderRadius: 1, boxShadow: 1 }}>
          <Tabs value={selectedStepIndex} onChange={handleTabChange}>
            {form.steps.map((step, index) => (
              <Tab
                key={step.id}
                label={step.name}
                sx={{ textTransform: 'none' }}
              />
            ))}
            <Tab icon={<AddIcon />} sx={{ minWidth: 40 }} />
          </Tabs>
        </Box>
      )}

      {/* Form preview container - constrained by device preview */}
      <Box
        sx={{
          width: '100%',
          maxWidth: deviceConfig.maxWidth,
          transition: 'max-width 0.3s ease',
        }}
      >
        <Box
          sx={{
            width: form.settings.width,
            maxWidth: form.settings.maxWidth,
            backgroundColor: form.settings.backgroundColor,
            borderRadius: `${form.settings.borderRadius}px`,
            padding: `${form.settings.padding.top}px ${form.settings.padding.right}px ${form.settings.padding.bottom}px ${form.settings.padding.left}px`,
            boxShadow: 3,
            minHeight: 200,
            margin: '0 auto',
          }}
        >
        {currentStep.blocks.length === 0 ? (
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              minHeight: 200,
              color: '#9ca3af',
              border: '2px dashed #d1d5db',
              borderRadius: 1,
              p: 3,
            }}
          >
            <Typography variant="body1" gutterBottom>
              Click blocks in the sidebar to add them
            </Typography>
            <Typography variant="body2">
              Start building your form by clicking blocks on the left
            </Typography>
          </Box>
        ) : (
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
            {currentStep.blocks.map((block) => (
              <SortableBlock
                key={block.id}
                block={block}
                isSelected={selectedBlockId === block.id}
                onSelect={() => selectBlock(block.id)}
                onDelete={() => removeBlock(block.id)}
                onDuplicate={() => duplicateBlock(block.id)}
              />
            ))}
          </Box>
        )}
        </Box>
      </Box>

      {/* Form type indicator */}
      <Box sx={{ mt: 2 }}>
        <Typography variant="caption" color="text.secondary">
          Form Type: {form.formType.charAt(0).toUpperCase() + form.formType.slice(1)} |
          Status: {form.status.charAt(0).toUpperCase() + form.status.slice(1)} |
          Preview: {deviceConfig.label}
        </Typography>
      </Box>
    </Box>
  );
};
