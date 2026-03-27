import type {ComponentType, ReactElement, ReactNode} from 'react';

declare global {
    const ReactBootstrap: {
        OverlayTrigger: ComponentType<{
            placement?: 'top' | 'bottom' | 'left' | 'right';
            delay?: {show?: number; hide?: number};
            overlay: ReactElement;
            children: ReactNode;
        }>;
        Tooltip: ComponentType<{id?: string; children?: ReactNode}>;
    };
}

export {};
