import type { VNode } from 'vue';
import BaseWrapper from './baseWrapper';
import type { RefSelector } from './types';
export declare class DOMWrapper<NodeType extends Node> extends BaseWrapper<NodeType> {
    protected readonly subTree: VNode | null | undefined;
    constructor(element: NodeType | null | undefined, subTree?: VNode | null);
    getRootNodes(): any[];
    getCurrentComponent(): import("vue").ComponentInternalInstance | undefined;
    find<K extends keyof HTMLElementTagNameMap>(selector: K): DOMWrapper<HTMLElementTagNameMap[K]>;
    find<K extends keyof SVGElementTagNameMap>(selector: K): DOMWrapper<SVGElementTagNameMap[K]>;
    find<T extends Element = Element>(selector: string): DOMWrapper<T>;
    find<T extends Node = Node>(selector: string | RefSelector): DOMWrapper<T>;
    findAll<K extends keyof HTMLElementTagNameMap>(selector: K): DOMWrapper<HTMLElementTagNameMap[K]>[];
    findAll<K extends keyof SVGElementTagNameMap>(selector: K): DOMWrapper<SVGElementTagNameMap[K]>[];
    findAll<T extends Element>(selector: string): DOMWrapper<T>[];
    findAllComponents(selector: any): any;
    private setChecked;
    setValue(value?: any): Promise<void>;
    private setSelected;
}
