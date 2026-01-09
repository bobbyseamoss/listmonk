import React, { useEffect } from 'react';
import { ThemeProvider, createTheme, CssBaseline, Box } from '@mui/material';
import { DndContext, closestCenter, DragEndEvent, DragStartEvent, DragOverlay } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy } from '@dnd-kit/sortable';
import { useFormBuilderStore } from '@/store';
import { BlocksSidebar } from './BlocksSidebar';
import { BuilderCanvas } from './BuilderCanvas';
import { PropertiesPanel } from './PropertiesPanel';
import { Toolbar } from './Toolbar';
import { BlockRenderer } from '@/blocks/BlockRenderer';
import type { FormConfig } from '@/types';

const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#2563eb',
    },
  },
  typography: {
    fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
  },
});

interface FormBuilderAppProps {
  initialForm?: FormConfig;
  onSave?: (form: FormConfig) => void;
  onCancel?: () => void;
}

export const FormBuilderApp: React.FC<FormBuilderAppProps> = ({ initialForm, onSave, onCancel }) => {
  const { form, setForm, selectedBlockId, selectedStepIndex, moveBlock, selectBlock } = useFormBuilderStore();
  const [activeDragId, setActiveDragId] = React.useState<string | null>(null);

  useEffect(() => {
    if (initialForm) {
      setForm(initialForm);
    }
  }, [initialForm, setForm]);

  // Listen for messages from parent (Vue)
  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      if (event.data.type === 'FORM_BUILDER_INIT') {
        setForm(event.data.form);
      } else if (event.data.type === 'FORM_BUILDER_REQUEST_SAVE') {
        window.parent.postMessage({ type: 'FORM_BUILDER_SAVE', form }, '*');
      }
    };

    window.addEventListener('message', handleMessage);

    // Notify parent that we're ready
    window.parent.postMessage({ type: 'FORM_BUILDER_READY' }, '*');

    return () => window.removeEventListener('message', handleMessage);
  }, [form, setForm]);

  const handleDragStart = (event: DragStartEvent) => {
    setActiveDragId(event.active.id as string);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveDragId(null);

    if (over && active.id !== over.id) {
      const currentStep = form.steps[selectedStepIndex];
      const oldIndex = currentStep.blocks.findIndex((b) => b.id === active.id);
      const newIndex = currentStep.blocks.findIndex((b) => b.id === over.id);

      if (oldIndex !== -1 && newIndex !== -1) {
        moveBlock(active.id as string, newIndex);
      }
    }
  };

  const handleSave = () => {
    if (onSave) {
      onSave(form);
    }
    window.parent.postMessage({ type: 'FORM_BUILDER_SAVE', form }, '*');
  };

  const handleCancel = () => {
    if (onCancel) {
      onCancel();
    }
    window.parent.postMessage({ type: 'FORM_BUILDER_CANCEL' }, '*');
  };

  const currentStep = form.steps[selectedStepIndex];
  const activeBlock = activeDragId ? currentStep.blocks.find((b) => b.id === activeDragId) : null;

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', overflow: 'hidden' }}>
        <Toolbar onSave={handleSave} onCancel={handleCancel} />
        <Box sx={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
          <DndContext
            collisionDetection={closestCenter}
            onDragStart={handleDragStart}
            onDragEnd={handleDragEnd}
          >
            <BlocksSidebar />
            <Box
              sx={{
                flex: 1,
                display: 'flex',
                flexDirection: 'column',
                backgroundColor: '#f5f5f5',
                overflow: 'auto',
                p: 3,
              }}
              onClick={(e) => {
                if (e.target === e.currentTarget) {
                  selectBlock(null);
                }
              }}
            >
              <SortableContext
                items={currentStep.blocks.map((b) => b.id)}
                strategy={verticalListSortingStrategy}
              >
                <BuilderCanvas />
              </SortableContext>
            </Box>
            <DragOverlay>
              {activeBlock ? (
                <Box
                  sx={{
                    backgroundColor: '#fff',
                    borderRadius: 1,
                    boxShadow: 3,
                    p: 1,
                    opacity: 0.9,
                  }}
                >
                  <BlockRenderer block={activeBlock} />
                </Box>
              ) : null}
            </DragOverlay>
            <PropertiesPanel />
          </DndContext>
        </Box>
      </Box>
    </ThemeProvider>
  );
};

export default FormBuilderApp;
