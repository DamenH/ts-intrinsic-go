//// [tests/cases/compiler/intrinsic/openapi.ts] ////

//// [openapi.ts]
// Resolve a $ref pointer like "#/components/schemas/Widget"
export const resolve = (spec: any, node: any) => {
    let cur: any = node;
    while (cur != undefined && cur['$ref'] != undefined) {
        let ref: string = cur['$ref'];
        let parts = ref.slice(2).split('/');
        let target: any = spec;
        for (let p of parts) {
            if (target == undefined) return undefined;
            target = target[p];
        }
        cur = target;
    }
    return cur;
};

// Convert a primitive/enum schema to a shape descriptor.
export const leafShape = (spec: any, rawSchema: any) => {
    let schema = resolve(spec, rawSchema);
    if (schema == undefined) return "any";
    let t = schema['type'];
    if (t == 'string') {
        let en = schema['enum'];
        if (en != undefined) {
            let result: Record<string, any> = {};
            result['_enum'] = en;
            return result;
        }
        return "string";
    }
    if (t == 'integer' || t == 'number') return "number";
    if (t == 'boolean') return "boolean";
    return "any";
};

// Convert an object schema to a property-shape map.
export const objectShape = (spec: any, rawSchema: any, forInput: any) => {
    let schema = resolve(spec, rawSchema);
    if (schema == undefined) return "any";
    if (schema['type'] != 'object') return leafShape(spec, rawSchema);
    let props = schema['properties'];
    if (props == undefined) return {};
    let result: Record<string, any> = {};
    for (let k of Object.keys(props)) {
        let ps = resolve(spec, props[k]);
        if (forInput == true && ps != undefined && ps['readOnly'] == true) { }
        else {
            result[k] = leafShape(spec, props[k]);
        }
    }
    return result;
};

// Convert any schema to a shape. Dispatches to leafShape/objectShape, handles arrays.
export const schemaToShape = (spec: any, rawSchema: any, forInput: any) => {
    let schema = resolve(spec, rawSchema);
    if (schema == undefined) return "any";
    let t = schema['type'];
    if (t == 'array') {
        let r: Record<string, any> = {};
        r['_array'] = objectShape(spec, schema['items'], false);
        return r;
    }
    if (t == 'object') return objectShape(spec, rawSchema, forInput);
    return leafShape(spec, rawSchema);
};

// Build paths descriptor: { "/path": { get: { params?, body?, data?, error? }, ... }, ... }
export const pathsType = (spec: any) => {
    if (typeof spec != 'object') return void { error: "Expected an OpenAPI spec" };
    let specPaths = spec['paths'];
    if (specPaths == undefined) return void { error: "No paths defined" };
    let result: Record<string, any> = {};
    for (let path of Object.keys(specPaths)) {
        let pathItem = specPaths[path];
        let methods: Record<string, any> = {};
        for (let m of ['get', 'post', 'put', 'patch', 'delete']) {
            let op = pathItem[m];
            if (op == undefined) { }
            else {
                let entry: Record<string, any> = {};
                // Path parameters
                let params = op['parameters'];
                if (params != undefined) {
                    let pathP: Record<string, any> = {};
                    let hasP = false;
                    for (let param of params) {
                        if (param['in'] == 'path') {
                            pathP[param['name']] = leafShape(spec, param['schema']);
                            hasP = true;
                        }
                    }
                    if (hasP) {
                        let p: Record<string, any> = {};
                        p['path'] = pathP;
                        entry['params'] = p;
                    }
                }
                // Request body
                let rb = op['requestBody'];
                if (rb != undefined) {
                    let content = rb['content'];
                    if (content != undefined) {
                        let cts = Object.keys(content);
                        if (cts.length > 0) {
                            let bodySchema = resolve(spec, content[cts[0]]['schema']);
                            if (bodySchema != undefined) {
                                entry['body'] = objectShape(spec, content[cts[0]]['schema'], true);
                                if (bodySchema['required'] == undefined) entry['bodyPartial'] = true;
                            }
                        }
                    }
                }
                // Success response
                let responses = op['responses'];
                if (responses != undefined) {
                    for (let status of Object.keys(responses)) {
                        if (status == '200' || status == '201') {
                            let resp = responses[status];
                            let rc = resp['content'];
                            if (rc != undefined) {
                                let rcts = Object.keys(rc);
                                if (rcts.length > 0) {
                                    entry['data'] = schemaToShape(spec, rc[rcts[0]]['schema'], false);
                                }
                            }
                        }
                    }
                    // Error response (from "default")
                    let errResp = responses['default'];
                    if (errResp != undefined) {
                        let erc = errResp['content'];
                        if (erc != undefined) {
                            let ercts = Object.keys(erc);
                            if (ercts.length > 0) {
                                entry['error'] = schemaToShape(spec, erc[ercts[0]]['schema'], false);
                            }
                        }
                    }
                }
                methods[m] = entry;
            }
        }
        result[path] = methods;
    }
    return result;
};

export type Paths<Spec> = Intrinsic<typeof pathsType, [Spec]>;

// Convert intrinsic shape descriptors to TypeScript types.
export type Widen<T> =
    T extends "string" ? string :
    T extends "number" ? number :
    T extends "boolean" ? boolean :
    T extends { _enum: infer E } ? E extends readonly (infer U)[] ? U : never :
    T extends { _array: infer I } ? Widen<I>[] :
    T extends object ? { [K in keyof T]: Widen<T[K]> } :
    T;

//// [openapi_userland.ts]
import { Paths, Widen } from "./openapi";

const spec = {
    openapi: "3.0.0",
    info: { title: "Widget Service", version: "0.0.0" },
    paths: {
        "/widgets": {
            get: {
                operationId: "Widgets_list",
                description: "List widgets",
                parameters: [],
                responses: {
                    "200": {
                        description: "The request has succeeded.",
                        content: { "application/json": { schema: { type: "array", items: { "$ref": "#/components/schemas/Widget" } } } }
                    },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                }
            },
            post: {
                operationId: "Widgets_create",
                description: "Create a widget",
                parameters: [],
                responses: {
                    "200": {
                        description: "The request has succeeded.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Widget" } } }
                    },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                },
                requestBody: {
                    required: true,
                    content: { "application/json": { schema: { "$ref": "#/components/schemas/Widget" } } }
                }
            }
        },
        "/widgets/{id}": {
            get: {
                operationId: "Widgets_read",
                description: "Read a widget",
                parameters: [{ name: "id", in: "path", required: true, schema: { type: "string", readOnly: true } }],
                responses: {
                    "200": {
                        description: "The request has succeeded.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Widget" } } }
                    },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                }
            },
            patch: {
                operationId: "Widgets_update",
                description: "Update a widget",
                parameters: [{ name: "id", in: "path", required: true, schema: { type: "string", readOnly: true } }],
                responses: {
                    "200": {
                        description: "The request has succeeded.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Widget" } } }
                    },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                },
                requestBody: {
                    required: true,
                    content: { "application/merge-patch+json": { schema: { "$ref": "#/components/schemas/WidgetMergePatchUpdate" } } }
                }
            },
            delete: {
                operationId: "Widgets_delete",
                description: "Delete a widget",
                parameters: [{ name: "id", in: "path", required: true, schema: { type: "string", readOnly: true } }],
                responses: {
                    "204": { description: "No content" },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                }
            }
        },
        "/widgets/{id}/analyze": {
            post: {
                operationId: "Widgets_analyze",
                description: "Analyze a widget",
                parameters: [{ name: "id", in: "path", required: true, schema: { type: "string", readOnly: true } }],
                responses: {
                    "200": {
                        description: "The request has succeeded.",
                        content: { "text/plain": { schema: { type: "string" } } }
                    },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                }
            }
        }
    },
    components: {
        schemas: {
            Widget: {
                type: "object",
                required: ["id", "weight", "color"],
                properties: {
                    id: { type: "string", readOnly: true },
                    weight: { type: "integer", format: "int32" },
                    color: { type: "string", enum: ["red", "blue"] }
                }
            },
            WidgetMergePatchUpdate: {
                type: "object",
                properties: {
                    weight: { type: "integer", format: "int32" },
                    color: { type: "string", enum: ["red", "blue"] }
                }
            },
            Error: {
                type: "object",
                required: ["code", "message"],
                properties: {
                    code: { type: "integer", format: "int32" },
                    message: { type: "string" }
                }
            }
        }
    }
} as const;

type SpecPaths = Paths<typeof spec>;

type WidgetsList = Widen<SpecPaths["/widgets"]["get"]["data"]>;
type WidgetsCreateBody = Widen<SpecPaths["/widgets"]["post"]["body"]>;
type WidgetsCreateData = Widen<SpecPaths["/widgets"]["post"]["data"]>;
type WidgetsCreateError = Widen<SpecPaths["/widgets"]["post"]["error"]>;

type WidgetReadParams = Widen<SpecPaths["/widgets/{id}"]["get"]["params"]>;
type WidgetReadData = Widen<SpecPaths["/widgets/{id}"]["get"]["data"]>;
type WidgetUpdateBody = Widen<SpecPaths["/widgets/{id}"]["patch"]["body"]>;
type WidgetAnalyzeData = Widen<SpecPaths["/widgets/{id}/analyze"]["post"]["data"]>;


//// [openapi.js]
// Resolve a $ref pointer like "#/components/schemas/Widget"
export const resolve = (spec, node) => {
    let cur = node;
    while (cur != undefined && cur['$ref'] != undefined) {
        let ref = cur['$ref'];
        let parts = ref.slice(2).split('/');
        let target = spec;
        for (let p of parts) {
            if (target == undefined)
                return undefined;
            target = target[p];
        }
        cur = target;
    }
    return cur;
};
// Convert a primitive/enum schema to a shape descriptor.
export const leafShape = (spec, rawSchema) => {
    let schema = resolve(spec, rawSchema);
    if (schema == undefined)
        return "any";
    let t = schema['type'];
    if (t == 'string') {
        let en = schema['enum'];
        if (en != undefined) {
            let result = {};
            result['_enum'] = en;
            return result;
        }
        return "string";
    }
    if (t == 'integer' || t == 'number')
        return "number";
    if (t == 'boolean')
        return "boolean";
    return "any";
};
// Convert an object schema to a property-shape map.
export const objectShape = (spec, rawSchema, forInput) => {
    let schema = resolve(spec, rawSchema);
    if (schema == undefined)
        return "any";
    if (schema['type'] != 'object')
        return leafShape(spec, rawSchema);
    let props = schema['properties'];
    if (props == undefined)
        return {};
    let result = {};
    for (let k of Object.keys(props)) {
        let ps = resolve(spec, props[k]);
        if (forInput == true && ps != undefined && ps['readOnly'] == true) { }
        else {
            result[k] = leafShape(spec, props[k]);
        }
    }
    return result;
};
// Convert any schema to a shape. Dispatches to leafShape/objectShape, handles arrays.
export const schemaToShape = (spec, rawSchema, forInput) => {
    let schema = resolve(spec, rawSchema);
    if (schema == undefined)
        return "any";
    let t = schema['type'];
    if (t == 'array') {
        let r = {};
        r['_array'] = objectShape(spec, schema['items'], false);
        return r;
    }
    if (t == 'object')
        return objectShape(spec, rawSchema, forInput);
    return leafShape(spec, rawSchema);
};
// Build paths descriptor: { "/path": { get: { params?, body?, data?, error? }, ... }, ... }
export const pathsType = (spec) => {
    if (typeof spec != 'object')
        return void { error: "Expected an OpenAPI spec" };
    let specPaths = spec['paths'];
    if (specPaths == undefined)
        return void { error: "No paths defined" };
    let result = {};
    for (let path of Object.keys(specPaths)) {
        let pathItem = specPaths[path];
        let methods = {};
        for (let m of ['get', 'post', 'put', 'patch', 'delete']) {
            let op = pathItem[m];
            if (op == undefined) { }
            else {
                let entry = {};
                // Path parameters
                let params = op['parameters'];
                if (params != undefined) {
                    let pathP = {};
                    let hasP = false;
                    for (let param of params) {
                        if (param['in'] == 'path') {
                            pathP[param['name']] = leafShape(spec, param['schema']);
                            hasP = true;
                        }
                    }
                    if (hasP) {
                        let p = {};
                        p['path'] = pathP;
                        entry['params'] = p;
                    }
                }
                // Request body
                let rb = op['requestBody'];
                if (rb != undefined) {
                    let content = rb['content'];
                    if (content != undefined) {
                        let cts = Object.keys(content);
                        if (cts.length > 0) {
                            let bodySchema = resolve(spec, content[cts[0]]['schema']);
                            if (bodySchema != undefined) {
                                entry['body'] = objectShape(spec, content[cts[0]]['schema'], true);
                                if (bodySchema['required'] == undefined)
                                    entry['bodyPartial'] = true;
                            }
                        }
                    }
                }
                // Success response
                let responses = op['responses'];
                if (responses != undefined) {
                    for (let status of Object.keys(responses)) {
                        if (status == '200' || status == '201') {
                            let resp = responses[status];
                            let rc = resp['content'];
                            if (rc != undefined) {
                                let rcts = Object.keys(rc);
                                if (rcts.length > 0) {
                                    entry['data'] = schemaToShape(spec, rc[rcts[0]]['schema'], false);
                                }
                            }
                        }
                    }
                    // Error response (from "default")
                    let errResp = responses['default'];
                    if (errResp != undefined) {
                        let erc = errResp['content'];
                        if (erc != undefined) {
                            let ercts = Object.keys(erc);
                            if (ercts.length > 0) {
                                entry['error'] = schemaToShape(spec, erc[ercts[0]]['schema'], false);
                            }
                        }
                    }
                }
                methods[m] = entry;
            }
        }
        result[path] = methods;
    }
    return result;
};
//// [openapi_userland.js]
const spec = {
    openapi: "3.0.0",
    info: { title: "Widget Service", version: "0.0.0" },
    paths: {
        "/widgets": {
            get: {
                operationId: "Widgets_list",
                description: "List widgets",
                parameters: [],
                responses: {
                    "200": {
                        description: "The request has succeeded.",
                        content: { "application/json": { schema: { type: "array", items: { "$ref": "#/components/schemas/Widget" } } } }
                    },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                }
            },
            post: {
                operationId: "Widgets_create",
                description: "Create a widget",
                parameters: [],
                responses: {
                    "200": {
                        description: "The request has succeeded.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Widget" } } }
                    },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                },
                requestBody: {
                    required: true,
                    content: { "application/json": { schema: { "$ref": "#/components/schemas/Widget" } } }
                }
            }
        },
        "/widgets/{id}": {
            get: {
                operationId: "Widgets_read",
                description: "Read a widget",
                parameters: [{ name: "id", in: "path", required: true, schema: { type: "string", readOnly: true } }],
                responses: {
                    "200": {
                        description: "The request has succeeded.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Widget" } } }
                    },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                }
            },
            patch: {
                operationId: "Widgets_update",
                description: "Update a widget",
                parameters: [{ name: "id", in: "path", required: true, schema: { type: "string", readOnly: true } }],
                responses: {
                    "200": {
                        description: "The request has succeeded.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Widget" } } }
                    },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                },
                requestBody: {
                    required: true,
                    content: { "application/merge-patch+json": { schema: { "$ref": "#/components/schemas/WidgetMergePatchUpdate" } } }
                }
            },
            delete: {
                operationId: "Widgets_delete",
                description: "Delete a widget",
                parameters: [{ name: "id", in: "path", required: true, schema: { type: "string", readOnly: true } }],
                responses: {
                    "204": { description: "No content" },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                }
            }
        },
        "/widgets/{id}/analyze": {
            post: {
                operationId: "Widgets_analyze",
                description: "Analyze a widget",
                parameters: [{ name: "id", in: "path", required: true, schema: { type: "string", readOnly: true } }],
                responses: {
                    "200": {
                        description: "The request has succeeded.",
                        content: { "text/plain": { schema: { type: "string" } } }
                    },
                    "default": {
                        description: "An unexpected error response.",
                        content: { "application/json": { schema: { "$ref": "#/components/schemas/Error" } } }
                    }
                }
            }
        }
    },
    components: {
        schemas: {
            Widget: {
                type: "object",
                required: ["id", "weight", "color"],
                properties: {
                    id: { type: "string", readOnly: true },
                    weight: { type: "integer", format: "int32" },
                    color: { type: "string", enum: ["red", "blue"] }
                }
            },
            WidgetMergePatchUpdate: {
                type: "object",
                properties: {
                    weight: { type: "integer", format: "int32" },
                    color: { type: "string", enum: ["red", "blue"] }
                }
            },
            Error: {
                type: "object",
                required: ["code", "message"],
                properties: {
                    code: { type: "integer", format: "int32" },
                    message: { type: "string" }
                }
            }
        }
    }
};
export {};
