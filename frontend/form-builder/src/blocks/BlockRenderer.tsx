import React from 'react';
import type { Block } from '@/types';

// Block-specific renderers
const TextBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div
      style={{
        fontSize: p.fontSize,
        fontWeight: p.fontWeight,
        color: p.color,
        textAlign: p.textAlign,
        padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px`,
      }}
    >
      {p.content}
    </div>
  );
};

const HeadingBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  const Tag = `h${p.level}` as keyof JSX.IntrinsicElements;
  return (
    <Tag
      style={{
        fontSize: p.fontSize,
        fontWeight: p.fontWeight,
        color: p.color,
        textAlign: p.textAlign,
        padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px`,
        margin: 0,
      }}
    >
      {p.content}
    </Tag>
  );
};

const ImageBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div
      style={{
        textAlign: p.alignment,
        padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px`,
      }}
    >
      {p.src ? (
        <img src={p.src} alt={p.alt} style={{ width: p.width, height: p.height, maxWidth: '100%' }} />
      ) : (
        <div
          style={{
            width: '100%',
            height: '150px',
            backgroundColor: '#f3f4f6',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#6b7280',
            border: '2px dashed #d1d5db',
            borderRadius: '4px',
            cursor: 'pointer',
          }}
        >
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" style={{ marginBottom: '8px', opacity: 0.5 }}>
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <circle cx="8.5" cy="8.5" r="1.5"/>
            <polyline points="21 15 16 10 5 21"/>
          </svg>
          <span style={{ fontSize: '14px', fontWeight: 500 }}>Click to select</span>
          <span style={{ fontSize: '12px', marginTop: '4px', opacity: 0.7 }}>Enter image URL in Properties panel →</span>
        </div>
      )}
    </div>
  );
};

const EmailInputBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div style={{ padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px` }}>
      <label style={{ display: 'block', marginBottom: '4px', fontSize: p.fontSize, color: p.labelColor }}>
        {p.label} {p.required && <span style={{ color: 'red' }}>*</span>}
      </label>
      <input
        type="email"
        placeholder={p.placeholder}
        disabled
        style={{
          width: '100%',
          padding: '10px 12px',
          fontSize: p.fontSize,
          backgroundColor: p.inputBgColor,
          border: `1px solid ${p.inputBorderColor}`,
          borderRadius: p.borderRadius,
          color: p.inputTextColor,
          boxSizing: 'border-box',
        }}
      />
    </div>
  );
};

const TextInputBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div style={{ padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px` }}>
      <label style={{ display: 'block', marginBottom: '4px', fontSize: p.fontSize, color: p.labelColor }}>
        {p.label} {p.required && <span style={{ color: 'red' }}>*</span>}
      </label>
      <input
        type="text"
        placeholder={p.placeholder}
        disabled
        style={{
          width: '100%',
          padding: '10px 12px',
          fontSize: p.fontSize,
          backgroundColor: p.inputBgColor,
          border: `1px solid ${p.inputBorderColor}`,
          borderRadius: p.borderRadius,
          color: p.inputTextColor,
          boxSizing: 'border-box',
        }}
      />
    </div>
  );
};

const TextareaBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div style={{ padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px` }}>
      <label style={{ display: 'block', marginBottom: '4px', fontSize: p.fontSize, color: p.labelColor }}>
        {p.label} {p.required && <span style={{ color: 'red' }}>*</span>}
      </label>
      <textarea
        placeholder={p.placeholder}
        rows={p.rows}
        disabled
        style={{
          width: '100%',
          padding: '10px 12px',
          fontSize: p.fontSize,
          backgroundColor: p.inputBgColor,
          border: `1px solid ${p.inputBorderColor}`,
          borderRadius: p.borderRadius,
          color: p.inputTextColor,
          boxSizing: 'border-box',
          resize: 'none',
        }}
      />
    </div>
  );
};

const RadioGroupBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div style={{ padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px` }}>
      <label style={{ display: 'block', marginBottom: '8px', fontSize: p.fontSize, color: p.labelColor }}>
        {p.label} {p.required && <span style={{ color: 'red' }}>*</span>}
      </label>
      <div style={{ display: 'flex', flexDirection: p.layout === 'vertical' ? 'column' : 'row', gap: '8px' }}>
        {p.options.map((option: any, index: number) => (
          <label key={index} style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: p.fontSize }}>
            <input type="radio" name={p.name} disabled />
            {option.label}
          </label>
        ))}
      </div>
    </div>
  );
};

const CheckboxGroupBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div style={{ padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px` }}>
      <label style={{ display: 'block', marginBottom: '8px', fontSize: p.fontSize, color: p.labelColor }}>
        {p.label} {p.required && <span style={{ color: 'red' }}>*</span>}
      </label>
      <div style={{ display: 'flex', flexDirection: p.layout === 'vertical' ? 'column' : 'row', gap: '8px' }}>
        {p.options.map((option: any, index: number) => (
          <label key={index} style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: p.fontSize }}>
            <input type="checkbox" disabled />
            {option.label}
          </label>
        ))}
      </div>
    </div>
  );
};

const DropdownBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div style={{ padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px` }}>
      <label style={{ display: 'block', marginBottom: '4px', fontSize: p.fontSize, color: p.labelColor }}>
        {p.label} {p.required && <span style={{ color: 'red' }}>*</span>}
      </label>
      <select
        disabled
        style={{
          width: '100%',
          padding: '10px 12px',
          fontSize: p.fontSize,
          backgroundColor: p.inputBgColor,
          border: `1px solid ${p.inputBorderColor}`,
          borderRadius: p.borderRadius,
          color: p.inputTextColor,
        }}
      >
        <option value="">{p.placeholder}</option>
        {p.options.map((option: any, index: number) => (
          <option key={index} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
};

const DatePickerBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div style={{ padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px` }}>
      <label style={{ display: 'block', marginBottom: '4px', fontSize: p.fontSize, color: p.labelColor }}>
        {p.label} {p.required && <span style={{ color: 'red' }}>*</span>}
      </label>
      <input
        type="date"
        disabled
        style={{
          width: '100%',
          padding: '10px 12px',
          fontSize: p.fontSize,
          backgroundColor: p.inputBgColor,
          border: `1px solid ${p.inputBorderColor}`,
          borderRadius: p.borderRadius,
          color: p.inputTextColor,
          boxSizing: 'border-box',
        }}
      />
    </div>
  );
};

const HiddenFieldBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div
      style={{
        padding: '8px',
        backgroundColor: '#f3f4f6',
        border: '1px dashed #d1d5db',
        borderRadius: '4px',
        fontSize: '12px',
        color: '#6b7280',
      }}
    >
      Hidden: {p.name} = {p.source === 'static' ? p.value : `[${p.source}:${p.paramName}]`}
    </div>
  );
};

const ButtonBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div style={{ margin: `${p.margin.top}px ${p.margin.right}px ${p.margin.bottom}px ${p.margin.left}px` }}>
      <button
        disabled
        style={{
          width: p.fullWidth ? '100%' : 'auto',
          padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px`,
          fontSize: p.fontSize,
          fontWeight: p.fontWeight,
          color: p.textColor,
          backgroundColor: p.bgColor,
          border: 'none',
          borderRadius: p.borderRadius,
          cursor: 'pointer',
        }}
      >
        {p.text}
      </button>
    </div>
  );
};

const DividerBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <hr
      style={{
        margin: `${p.margin.top}px 0 ${p.margin.bottom}px 0`,
        border: 'none',
        borderTop: `${p.thickness}px ${p.style} ${p.color}`,
      }}
    />
  );
};

const SpacerBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return <div style={{ height: p.height }} />;
};

const CouponBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div
      style={{
        padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px`,
        backgroundColor: p.bgColor,
        border: `1px solid ${p.borderColor}`,
        borderRadius: p.borderRadius,
        textAlign: 'center',
      }}
    >
      <div style={{ fontSize: p.fontSize - 2, color: p.textColor, marginBottom: '4px' }}>{p.title}</div>
      <div style={{ fontSize: p.fontSize + 4, fontWeight: p.fontWeight, color: p.textColor, letterSpacing: '2px' }}>
        XXXXX-XXXXX
      </div>
    </div>
  );
};

const CountdownBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  return (
    <div
      style={{
        padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px`,
        backgroundColor: p.bgColor,
        borderRadius: p.borderRadius,
        display: 'flex',
        justifyContent: 'center',
        gap: '12px',
      }}
    >
      {p.showDays && (
        <div style={{ textAlign: 'center' }}>
          <div style={{ fontSize: p.fontSize, fontWeight: 'bold', color: p.textColor }}>00</div>
          <div style={{ fontSize: p.fontSize * 0.5, color: p.textColor }}>Days</div>
        </div>
      )}
      {p.showHours && (
        <div style={{ textAlign: 'center' }}>
          <div style={{ fontSize: p.fontSize, fontWeight: 'bold', color: p.textColor }}>00</div>
          <div style={{ fontSize: p.fontSize * 0.5, color: p.textColor }}>Hours</div>
        </div>
      )}
      {p.showMinutes && (
        <div style={{ textAlign: 'center' }}>
          <div style={{ fontSize: p.fontSize, fontWeight: 'bold', color: p.textColor }}>00</div>
          <div style={{ fontSize: p.fontSize * 0.5, color: p.textColor }}>Min</div>
        </div>
      )}
      {p.showSeconds && (
        <div style={{ textAlign: 'center' }}>
          <div style={{ fontSize: p.fontSize, fontWeight: 'bold', color: p.textColor }}>00</div>
          <div style={{ fontSize: p.fontSize * 0.5, color: p.textColor }}>Sec</div>
        </div>
      )}
    </div>
  );
};

const SignupCounterBlockRenderer: React.FC<{ props: Block['props'] }> = ({ props }) => {
  const p = props as any;
  const text = p.text.replace('{count}', `<span style="color: ${p.countColor}; font-weight: bold">${p.count.toLocaleString()}</span>`);
  return (
    <div
      style={{
        padding: `${p.padding.top}px ${p.padding.right}px ${p.padding.bottom}px ${p.padding.left}px`,
        fontSize: p.fontSize,
        color: p.textColor,
        textAlign: 'center',
      }}
      dangerouslySetInnerHTML={{ __html: text }}
    />
  );
};

// Main BlockRenderer component
interface BlockRendererProps {
  block: Block;
}

export const BlockRenderer: React.FC<BlockRendererProps> = ({ block }) => {
  const renderers: Record<string, React.FC<{ props: Block['props'] }>> = {
    text: TextBlockRenderer,
    heading: HeadingBlockRenderer,
    image: ImageBlockRenderer,
    'email-input': EmailInputBlockRenderer,
    'text-input': TextInputBlockRenderer,
    textarea: TextareaBlockRenderer,
    'radio-group': RadioGroupBlockRenderer,
    'checkbox-group': CheckboxGroupBlockRenderer,
    dropdown: DropdownBlockRenderer,
    'date-picker': DatePickerBlockRenderer,
    'hidden-field': HiddenFieldBlockRenderer,
    button: ButtonBlockRenderer,
    divider: DividerBlockRenderer,
    spacer: SpacerBlockRenderer,
    coupon: CouponBlockRenderer,
    countdown: CountdownBlockRenderer,
    'signup-counter': SignupCounterBlockRenderer,
  };

  const Renderer = renderers[block.type];
  if (!Renderer) {
    return <div>Unknown block type: {block.type}</div>;
  }

  return <Renderer props={block.props} />;
};
