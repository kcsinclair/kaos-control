import type { VTUVNodeTypeTransformer } from './util';
import type { Component, ConcreteComponent, KeepAlive, Teleport, VNodeTypes } from 'vue';
export type CustomCreateStub = (params: {
    name: string;
    component: ConcreteComponent;
    registerStub: (config: {
        source: Component;
        stub: Component;
    }) => void;
}) => ConcreteComponent;
interface StubOptions {
    name: string;
    type?: VNodeTypes | typeof Teleport | typeof KeepAlive;
    renderStubDefaultSlot?: boolean;
}
export declare const createStub: ({ name, type, renderStubDefaultSlot }: StubOptions) => import("vue").DefineComponent<{}, () => import("vue").VNode<import("vue").RendererNode, import("vue").RendererElement, {
    [key: string]: any;
}>, {}, {}, {}, import("vue").ComponentOptionsMixin, import("vue").ComponentOptionsMixin, {}, string, import("vue").PublicProps, Readonly<{}> & Readonly<{}>, {} | {
    [x: string]: any;
}, {}, {}, {}, string, import("vue").ComponentProvideOptions, true, {}, any>;
export interface CreateStubComponentsTransformerConfig {
    rootComponents: {
        component?: Component;
        functional?: Component;
    };
    stubs?: Record<string, Component | boolean>;
    shallow?: boolean;
    renderStubDefaultSlot: boolean;
}
export declare function createStubComponentsTransformer({ rootComponents, stubs, shallow, renderStubDefaultSlot }: CreateStubComponentsTransformerConfig): VTUVNodeTypeTransformer;
export {};
