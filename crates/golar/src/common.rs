#![allow(non_snake_case)]

use std::iter::FusedIterator;
use std::marker;
use std::slice;
use std::str;

use crate::ast_generated::*;
use crate::flags_generated::*;

type TypeId = u32;

#[repr(C)]
#[derive(Debug, Copy, Clone)]
pub struct TextRange {
    pub pos: i32,
    pub end: i32,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct GoSlice<T> {
    pub data: *const T,
    pub len: isize,
    pub cap: isize,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct GoItab {
    pub _inter: *const u8,
    pub _type: *const u8,
    pub hash: u32,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct GoIface {
    pub itab: *const GoItab,
    pub data: *const u8,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct GoString {
    pub data: *const u8,
    pub len: isize,
}

pub unsafe trait SliceItem<'a, Raw>: Sized {
    unsafe fn from_raw_unchecked(raw: *const Raw) -> Self;
}

#[derive(Copy, Clone)]
pub struct SliceIter<'a, Raw, Wrapped> {
    data: *const *const Raw,
    len: usize,
    index: usize,
    _marker: marker::PhantomData<(&'a (), Wrapped)>,
}

impl<'a, Raw, Wrapped> SliceIter<'a, Raw, Wrapped>
where
    Wrapped: SliceItem<'a, Raw>,
{
    #[inline(always)]
    pub fn new(raw: GoSlice<*const Raw>) -> Self {
        Self {
            data: raw.data,
            len: raw.len as usize,
            index: 0,
            _marker: marker::PhantomData,
        }
    }
}

impl<'a, Raw, Wrapped> Iterator for SliceIter<'a, Raw, Wrapped>
where
    Wrapped: SliceItem<'a, Raw>,
{
    type Item = Wrapped;

    #[inline(always)]
    fn next(&mut self) -> Option<Self::Item> {
        if self.index >= self.len {
            return None;
        }

        let raw = unsafe { *self.data.add(self.index) };
        self.index += 1;
        Some(unsafe { Wrapped::from_raw_unchecked(raw) })
    }

    #[inline(always)]
    fn size_hint(&self) -> (usize, Option<usize>) {
        let remaining = self.len();
        (remaining, Some(remaining))
    }
}

impl<'a, Raw, Wrapped> ExactSizeIterator for SliceIter<'a, Raw, Wrapped>
where
    Wrapped: SliceItem<'a, Raw>,
{
    #[inline(always)]
    fn len(&self) -> usize {
        self.len - self.index
    }
}

impl<'a, Raw, Wrapped> FusedIterator for SliceIter<'a, Raw, Wrapped> where
    Wrapped: SliceItem<'a, Raw>
{
}

#[inline(always)]
pub fn slice_iter<'a, Raw, Wrapped>(raw: GoSlice<*const Raw>) -> SliceIter<'a, Raw, Wrapped>
where
    Wrapped: SliceItem<'a, Raw>,
{
    SliceIter::new(raw)
}

impl GoString {
    #[inline(always)]
    pub unsafe fn as_bytes<'a>(&self) -> &'a [u8] {
        unsafe { slice::from_raw_parts(self.data, self.len as usize) }
    }

    #[inline(always)]
    pub unsafe fn as_str<'a>(&self) -> &'a str {
        unsafe { str::from_utf8_unchecked(self.as_bytes()) }
    }
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawNode {
    pub kind: Kind,
    pub flags: u32,
    pub loc: TextRange,
    // Upstream's atomic node ID is separate from Golar's per-file protocol ID.
    pub globalId: u64,
    pub parent: *const RawNode,
    pub sourceFileId: u32,
    pub id: u32,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawNodeList {
    pub loc: TextRange,
    pub nodes: GoSlice<*const RawNode>,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawModifierList {
    pub nodeList: RawNodeList,
    pub modifierFlags: ModifierFlags,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawSymbol {
    pub flags: SymbolFlags,
    pub checkFlags: CheckFlags,
    pub id: u64,
    pub data: *const RawSymbolData,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawSymbolData {
    pub name: GoString,
    pub declarations: GoSlice<*const RawNode>,
    pub valueDeclaration: *const RawNode,
    pub parent: *const RawSymbol,
    pub extra: *const RawSymbolExtra,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawSymbolExtra {
    pub members: *const u8,
    pub exports: *const u8,
    pub exportSymbol: *const RawSymbol,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawTypeAlias {
    pub symbol: *const RawSymbol,
    pub typeArguments: GoSlice<*const RawType>,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawType {
    pub flags: TypeFlags,
    pub objectFlags: ObjectFlags,
    pub id: TypeId,
    pub kind: u8,
    pub _pad0: [u8; 3],
    pub symbol: *const RawSymbol,
    pub alias: *const RawTypeAlias,
    pub checker: *const u8,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawCompositeSignature {
    pub isUnion: u8,
    pub _pad0: [u8; 7],
    pub signatures: GoSlice<*const RawSignature>,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawCheckerTypePredicate {
    pub kind: TypePredicateKind,
    pub parameterIndex: i32,
    pub parameterName: GoString,
    pub t: *const RawType,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawIndexInfo {
    pub keyType: *const RawType,
    pub valueType: *const RawType,
    pub isReadonly: u8,
    pub _pad0: [u8; 7],
    pub declaration: *const RawNode,
    pub indexSymbol: *const RawSymbol,
    pub components: GoSlice<*const RawNode>,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawTupleElementInfo {
    pub flags: ElementFlags,
    pub _pad0: [u8; 4],
    pub labeledDeclaration: *const RawNode,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawConditionalRoot {
    pub node: *const RawNode,
    pub checkType: *const RawType,
    pub extendsType: *const RawType,
    pub isDistributive: u8,
    pub _pad0: [u8; 7],
    pub inferTypeParameters: GoSlice<*const RawType>,
    pub outerTypeParameters: GoSlice<*const RawType>,
    pub instantiations: *const u8,
    pub alias: *const RawTypeAlias,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawSignature {
    pub id: u32,
    pub flags: SignatureFlags,
    pub minArgumentCount: i32,
    pub resolvedMinArgumentCount: i32,
    pub declaration: *const RawNode,
    pub parameters: GoSlice<*const RawSymbol>,
    pub resolvedReturnType: *const RawType,
    pub target: *const RawSignature,
    pub mapper: *const u8,
    pub extra: *const RawSignatureExtra,
}

#[repr(C)]
#[derive(Copy, Clone)]
pub struct RawSignatureExtra {
    pub typeParameters: GoSlice<*const RawType>,
    pub thisParameter: *const RawSymbol,
    pub resolvedTypePredicate: *const RawCheckerTypePredicate,
    pub isolatedSignatureType: *const RawType,
    pub composite: *const RawCompositeSignature,
}

pub trait FromNode<'a>: Sized {
    fn matches(kind: Kind) -> bool;
    unsafe fn from_node_unchecked(node: Node<'a>) -> Self;
}

pub trait FromType<'a>: Sized {
    fn matches(flags: TypeFlags, object_flags: ObjectFlags) -> bool;
    unsafe fn from_type_unchecked(t: Type<'a>) -> Self;
}

#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct NodeList<'a> {
    raw: *const RawNodeList,
    _marker: marker::PhantomData<&'a ()>,
}

impl<'a> NodeList<'a> {
    #[inline(always)]
    pub fn from_raw(raw: *const RawNodeList) -> Self {
        Self {
            raw,
            _marker: marker::PhantomData,
        }
    }

    #[inline(always)]
    pub fn len(self) -> usize {
        unsafe { (*self.raw).nodes.len as usize }
    }

    #[inline(always)]
    pub fn is_empty(self) -> bool {
        self.len() == 0
    }

    #[inline(always)]
    pub fn get(self, idx: usize) -> Option<Node<'a>> {
        unsafe {
            if idx < (*self.raw).nodes.len as usize {
                node_from_raw(*(*self.raw).nodes.data.add(idx))
            } else {
                None
            }
        }
    }
}

#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct ModifierList<'a> {
    raw: *const RawModifierList,
    _marker: marker::PhantomData<&'a ()>,
}

impl<'a> ModifierList<'a> {
    #[inline(always)]
    pub fn from_raw(raw: *const RawModifierList) -> Self {
        Self {
            raw,
            _marker: marker::PhantomData,
        }
    }

    #[inline(always)]
    pub fn as_node_list(self) -> NodeList<'a> {
        NodeList::from_raw(self.raw.cast())
    }

    #[inline(always)]
    pub fn modifier_flags(self) -> ModifierFlags {
        unsafe { (*self.raw).modifierFlags }
    }

    #[inline(always)]
    pub fn len(self) -> usize {
        self.as_node_list().len()
    }

    #[inline(always)]
    pub fn get(self, idx: usize) -> Option<Node<'a>> {
        self.as_node_list().get(idx)
    }
}

#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct Node<'a> {
    pub(crate) raw: *const RawNode,
    _marker: marker::PhantomData<&'a ()>,
}

impl<'a> Node<'a> {
    #[inline(always)]
    pub fn from_raw(raw: *const RawNode) -> Self {
        Self {
            raw,
            _marker: marker::PhantomData,
        }
    }

    #[inline(always)]
    pub fn kind(self) -> Kind {
        unsafe { (*self.raw).kind }
    }

    #[inline(always)]
    pub fn pos(self) -> i32 {
        unsafe { (*self.raw).loc.pos }
    }

    #[inline(always)]
    pub fn end(self) -> i32 {
        unsafe { (*self.raw).loc.end }
    }

    #[inline(always)]
    pub fn parent(self) -> Option<Node<'a>> {
        node_from_raw(unsafe { (*self.raw).parent })
    }

    #[inline(always)]
    pub fn source_file_id(self) -> u32 {
        unsafe { (*self.raw).sourceFileId }
    }

    #[inline(always)]
    pub fn id(self) -> u32 {
        unsafe { (*self.raw).id }
    }

    #[inline(always)]
    pub fn cast<T: FromNode<'a>>(self) -> Option<T> {
        if T::matches(self.kind()) {
            Some(unsafe { T::from_node_unchecked(self) })
        } else {
            None
        }
    }

    #[inline(always)]
    pub fn for_each_child<F>(self, f: &mut F) -> bool
    where
        F: FnMut(Node<'a>) -> bool,
    {
        for_each_child(f, self)
    }
}

#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct Symbol<'a> {
    raw: *const RawSymbol,
    _marker: marker::PhantomData<&'a ()>,
}

impl<'a> Symbol<'a> {
    #[inline(always)]
    pub fn from_raw(raw: *const RawSymbol) -> Self {
        Self {
            raw,
            _marker: marker::PhantomData,
        }
    }

    #[inline(always)]
    pub fn flags(self) -> SymbolFlags {
        unsafe { (*self.raw).flags }
    }

    #[inline(always)]
    pub fn check_flags(self) -> CheckFlags {
        unsafe { (*self.raw).checkFlags }
    }

    #[inline(always)]
    pub fn name(self) -> &'a str {
        unsafe { (*(*self.raw).data).name.as_str() }
    }

    #[inline(always)]
    pub fn declarations(self) -> SliceIter<'a, RawNode, Node<'a>> {
        slice_iter(unsafe { (*(*self.raw).data).declarations })
    }

    #[inline(always)]
    pub fn value_declaration(self) -> Option<Node<'a>> {
        node_from_raw(unsafe { (*(*self.raw).data).valueDeclaration })
    }

    #[inline(always)]
    pub fn parent(self) -> Option<Symbol<'a>> {
        symbol_from_raw(unsafe { (*(*self.raw).data).parent })
    }

    #[inline(always)]
    pub fn export_symbol(self) -> Option<Symbol<'a>> {
        unsafe {
            let extra = (*(*self.raw).data).extra;
            if extra.is_null() {
                None
            } else {
                symbol_from_raw((*extra).exportSymbol)
            }
        }
    }
}

#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct TypeAlias<'a> {
    raw: *const RawTypeAlias,
    _marker: marker::PhantomData<&'a ()>,
}

impl<'a> TypeAlias<'a> {
    #[inline(always)]
    pub fn from_raw(raw: *const RawTypeAlias) -> Self {
        Self {
            raw,
            _marker: marker::PhantomData,
        }
    }

    #[inline(always)]
    pub fn symbol(self) -> Option<Symbol<'a>> {
        symbol_from_raw(unsafe { (*self.raw).symbol })
    }

    #[inline(always)]
    pub fn type_arguments(self) -> SliceIter<'a, RawType, Type<'a>> {
        slice_iter(unsafe { (*self.raw).typeArguments })
    }
}

#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct Type<'a> {
    pub(crate) raw: *const RawType,
    _marker: marker::PhantomData<&'a ()>,
}

impl<'a> Type<'a> {
    #[inline(always)]
    pub fn from_raw(raw: *const RawType) -> Self {
        Self {
            raw,
            _marker: marker::PhantomData,
        }
    }

    #[inline(always)]
    pub fn flags(self) -> TypeFlags {
        unsafe { (*self.raw).flags }
    }

    #[inline(always)]
    pub fn object_flags(self) -> ObjectFlags {
        unsafe { (*self.raw).objectFlags }
    }

    #[inline(always)]
    pub fn id(self) -> TypeId {
        unsafe { (*self.raw).id }
    }

    #[inline(always)]
    pub fn symbol(self) -> Option<Symbol<'a>> {
        symbol_from_raw(unsafe { (*self.raw).symbol })
    }

    #[inline(always)]
    pub fn alias(self) -> Option<TypeAlias<'a>> {
        type_alias_from_raw(unsafe { (*self.raw).alias })
    }

    #[inline(always)]
    pub fn cast<T: FromType<'a>>(self) -> Option<T> {
        if T::matches(self.flags(), self.object_flags()) {
            Some(unsafe { T::from_type_unchecked(self) })
        } else {
            None
        }
    }
}

#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct CompositeSignature<'a> {
    raw: *const RawCompositeSignature,
    _marker: marker::PhantomData<&'a ()>,
}

impl<'a> CompositeSignature<'a> {
    #[inline(always)]
    pub fn from_raw(raw: *const RawCompositeSignature) -> Self {
        Self {
            raw,
            _marker: marker::PhantomData,
        }
    }

    #[inline(always)]
    pub fn is_union(self) -> bool {
        unsafe { (*self.raw).isUnion != 0 }
    }

    #[inline(always)]
    pub fn signatures(self) -> SliceIter<'a, RawSignature, Signature<'a>> {
        slice_iter(unsafe { (*self.raw).signatures })
    }
}

#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct CheckerTypePredicate<'a> {
    raw: *const RawCheckerTypePredicate,
    _marker: marker::PhantomData<&'a ()>,
}

impl<'a> CheckerTypePredicate<'a> {
    #[inline(always)]
    pub fn from_raw(raw: *const RawCheckerTypePredicate) -> Self {
        Self {
            raw,
            _marker: marker::PhantomData,
        }
    }

    #[inline(always)]
    pub fn kind(self) -> TypePredicateKind {
        unsafe { (*self.raw).kind }
    }

    #[inline(always)]
    pub fn parameter_index(self) -> i32 {
        unsafe { (*self.raw).parameterIndex }
    }

    #[inline(always)]
    pub fn parameter_name(self) -> &'a str {
        unsafe { (*self.raw).parameterName.as_str() }
    }

    #[inline(always)]
    pub fn type_(self) -> Option<Type<'a>> {
        type_from_raw(unsafe { (*self.raw).t })
    }
}

#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct IndexInfo<'a> {
    raw: *const RawIndexInfo,
    _marker: marker::PhantomData<&'a ()>,
}

impl<'a> IndexInfo<'a> {
    #[inline(always)]
    pub fn from_raw(raw: *const RawIndexInfo) -> Self {
        Self {
            raw,
            _marker: marker::PhantomData,
        }
    }

    #[inline(always)]
    pub fn key_type(self) -> Option<Type<'a>> {
        type_from_raw(unsafe { (*self.raw).keyType })
    }

    #[inline(always)]
    pub fn value_type(self) -> Option<Type<'a>> {
        type_from_raw(unsafe { (*self.raw).valueType })
    }

    #[inline(always)]
    pub fn is_readonly(self) -> bool {
        unsafe { (*self.raw).isReadonly != 0 }
    }

    #[inline(always)]
    pub fn declaration(self) -> Option<Node<'a>> {
        node_from_raw(unsafe { (*self.raw).declaration })
    }

    #[inline(always)]
    pub fn index_symbol(self) -> Option<Symbol<'a>> {
        symbol_from_raw(unsafe { (*self.raw).indexSymbol })
    }

    #[inline(always)]
    pub fn components(self) -> SliceIter<'a, RawNode, Node<'a>> {
        slice_iter(unsafe { (*self.raw).components })
    }
}

#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct ConditionalRoot<'a> {
    raw: *const RawConditionalRoot,
    _marker: marker::PhantomData<&'a ()>,
}

impl<'a> ConditionalRoot<'a> {
    #[inline(always)]
    pub fn from_raw(raw: *const RawConditionalRoot) -> Self {
        Self {
            raw,
            _marker: marker::PhantomData,
        }
    }

    #[inline(always)]
    pub fn node(self) -> Option<Node<'a>> {
        node_from_raw(unsafe { (*self.raw).node })
    }

    #[inline(always)]
    pub fn check_type(self) -> Option<Type<'a>> {
        type_from_raw(unsafe { (*self.raw).checkType })
    }

    #[inline(always)]
    pub fn extends_type(self) -> Option<Type<'a>> {
        type_from_raw(unsafe { (*self.raw).extendsType })
    }

    #[inline(always)]
    pub fn is_distributive(self) -> bool {
        unsafe { (*self.raw).isDistributive != 0 }
    }

    #[inline(always)]
    pub fn infer_type_parameters(self) -> SliceIter<'a, RawType, Type<'a>> {
        slice_iter(unsafe { (*self.raw).inferTypeParameters })
    }

    #[inline(always)]
    pub fn outer_type_parameters(self) -> SliceIter<'a, RawType, Type<'a>> {
        slice_iter(unsafe { (*self.raw).outerTypeParameters })
    }

    #[inline(always)]
    pub fn alias(self) -> Option<TypeAlias<'a>> {
        type_alias_from_raw(unsafe { (*self.raw).alias })
    }
}

#[repr(transparent)]
#[derive(Copy, Clone)]
pub struct Signature<'a> {
    raw: *const RawSignature,
    _marker: marker::PhantomData<&'a ()>,
}

impl<'a> Signature<'a> {
    #[inline(always)]
    pub fn from_raw(raw: *const RawSignature) -> Self {
        Self {
            raw,
            _marker: marker::PhantomData,
        }
    }

    #[inline(always)]
    pub fn flags(self) -> SignatureFlags {
        unsafe { (*self.raw).flags }
    }

    #[inline(always)]
    pub fn min_argument_count(self) -> i32 {
        unsafe { (*self.raw).minArgumentCount }
    }

    #[inline(always)]
    pub fn resolved_min_argument_count(self) -> i32 {
        unsafe { (*self.raw).resolvedMinArgumentCount }
    }

    #[inline(always)]
    pub fn declaration(self) -> Option<Node<'a>> {
        node_from_raw(unsafe { (*self.raw).declaration })
    }

    #[inline(always)]
    pub fn type_parameters(self) -> SliceIter<'a, RawType, Type<'a>> {
        unsafe {
            let extra = (*self.raw).extra;
            if extra.is_null() {
                slice_iter(GoSlice {
                    data: std::ptr::null(),
                    len: 0,
                    cap: 0,
                })
            } else {
                slice_iter((*extra).typeParameters)
            }
        }
    }

    #[inline(always)]
    pub fn parameters(self) -> SliceIter<'a, RawSymbol, Symbol<'a>> {
        slice_iter(unsafe { (*self.raw).parameters })
    }

    #[inline(always)]
    pub fn this_parameter(self) -> Option<Symbol<'a>> {
        unsafe {
            let extra = (*self.raw).extra;
            if extra.is_null() {
                None
            } else {
                symbol_from_raw((*extra).thisParameter)
            }
        }
    }

    #[inline(always)]
    pub fn resolved_return_type(self) -> Option<Type<'a>> {
        type_from_raw(unsafe { (*self.raw).resolvedReturnType })
    }

    #[inline(always)]
    pub fn resolved_type_predicate(self) -> Option<CheckerTypePredicate<'a>> {
        unsafe {
            let extra = (*self.raw).extra;
            if extra.is_null() {
                None
            } else {
                type_predicate_from_raw((*extra).resolvedTypePredicate)
            }
        }
    }

    #[inline(always)]
    pub fn target(self) -> Option<Signature<'a>> {
        signature_from_raw(unsafe { (*self.raw).target })
    }

    #[inline(always)]
    pub fn has_rest_parameter(self) -> bool {
        self.flags().contains(SignatureFlags::HAS_REST_PARAMETER)
    }

    #[inline(always)]
    pub fn isolated_signature_type(self) -> Option<Type<'a>> {
        unsafe {
            let extra = (*self.raw).extra;
            if extra.is_null() {
                None
            } else {
                type_from_raw((*extra).isolatedSignatureType)
            }
        }
    }

    #[inline(always)]
    pub fn composite(self) -> Option<CompositeSignature<'a>> {
        unsafe {
            let extra = (*self.raw).extra;
            if extra.is_null() {
                None
            } else {
                composite_signature_from_raw((*extra).composite)
            }
        }
    }
}

unsafe impl<'a> SliceItem<'a, RawNode> for Node<'a> {
    #[inline(always)]
    unsafe fn from_raw_unchecked(raw: *const RawNode) -> Self {
        Self::from_raw(raw)
    }
}

unsafe impl<'a> SliceItem<'a, RawNodeList> for NodeList<'a> {
    #[inline(always)]
    unsafe fn from_raw_unchecked(raw: *const RawNodeList) -> Self {
        Self::from_raw(raw)
    }
}

unsafe impl<'a> SliceItem<'a, RawModifierList> for ModifierList<'a> {
    #[inline(always)]
    unsafe fn from_raw_unchecked(raw: *const RawModifierList) -> Self {
        Self::from_raw(raw)
    }
}

unsafe impl<'a> SliceItem<'a, RawSymbol> for Symbol<'a> {
    #[inline(always)]
    unsafe fn from_raw_unchecked(raw: *const RawSymbol) -> Self {
        Self::from_raw(raw)
    }
}

unsafe impl<'a> SliceItem<'a, RawTypeAlias> for TypeAlias<'a> {
    #[inline(always)]
    unsafe fn from_raw_unchecked(raw: *const RawTypeAlias) -> Self {
        Self::from_raw(raw)
    }
}

unsafe impl<'a> SliceItem<'a, RawType> for Type<'a> {
    #[inline(always)]
    unsafe fn from_raw_unchecked(raw: *const RawType) -> Self {
        Self::from_raw(raw)
    }
}

unsafe impl<'a> SliceItem<'a, RawCompositeSignature> for CompositeSignature<'a> {
    #[inline(always)]
    unsafe fn from_raw_unchecked(raw: *const RawCompositeSignature) -> Self {
        Self::from_raw(raw)
    }
}

unsafe impl<'a> SliceItem<'a, RawCheckerTypePredicate> for CheckerTypePredicate<'a> {
    #[inline(always)]
    unsafe fn from_raw_unchecked(raw: *const RawCheckerTypePredicate) -> Self {
        Self::from_raw(raw)
    }
}

unsafe impl<'a> SliceItem<'a, RawIndexInfo> for IndexInfo<'a> {
    #[inline(always)]
    unsafe fn from_raw_unchecked(raw: *const RawIndexInfo) -> Self {
        Self::from_raw(raw)
    }
}

unsafe impl<'a> SliceItem<'a, RawConditionalRoot> for ConditionalRoot<'a> {
    #[inline(always)]
    unsafe fn from_raw_unchecked(raw: *const RawConditionalRoot) -> Self {
        Self::from_raw(raw)
    }
}

unsafe impl<'a> SliceItem<'a, RawSignature> for Signature<'a> {
    #[inline(always)]
    unsafe fn from_raw_unchecked(raw: *const RawSignature) -> Self {
        Self::from_raw(raw)
    }
}

#[inline(always)]
pub fn node_from_raw<'a>(raw: *const RawNode) -> Option<Node<'a>> {
    if raw.is_null() {
        None
    } else {
        Some(Node::from_raw(raw))
    }
}

#[inline(always)]
pub fn node_list_from_raw<'a>(raw: *const RawNodeList) -> Option<NodeList<'a>> {
    if raw.is_null() {
        None
    } else {
        Some(NodeList::from_raw(raw))
    }
}

#[inline(always)]
pub fn modifier_list_from_raw<'a>(raw: *const RawModifierList) -> Option<ModifierList<'a>> {
    if raw.is_null() {
        None
    } else {
        Some(ModifierList::from_raw(raw))
    }
}

#[inline(always)]
pub fn symbol_from_raw<'a>(raw: *const RawSymbol) -> Option<Symbol<'a>> {
    if raw.is_null() {
        None
    } else {
        Some(Symbol::from_raw(raw))
    }
}

#[inline(always)]
pub fn type_alias_from_raw<'a>(raw: *const RawTypeAlias) -> Option<TypeAlias<'a>> {
    if raw.is_null() {
        None
    } else {
        Some(TypeAlias::from_raw(raw))
    }
}

#[inline(always)]
pub fn type_from_raw<'a>(raw: *const RawType) -> Option<Type<'a>> {
    if raw.is_null() {
        None
    } else {
        Some(Type::from_raw(raw))
    }
}

#[inline(always)]
pub fn composite_signature_from_raw<'a>(
    raw: *const RawCompositeSignature,
) -> Option<CompositeSignature<'a>> {
    if raw.is_null() {
        None
    } else {
        Some(CompositeSignature::from_raw(raw))
    }
}

#[inline(always)]
pub fn signature_from_raw<'a>(raw: *const RawSignature) -> Option<Signature<'a>> {
    if raw.is_null() {
        None
    } else {
        Some(Signature::from_raw(raw))
    }
}

#[inline(always)]
pub fn type_predicate_from_raw<'a>(
    raw: *const RawCheckerTypePredicate,
) -> Option<CheckerTypePredicate<'a>> {
    if raw.is_null() {
        None
    } else {
        Some(CheckerTypePredicate::from_raw(raw))
    }
}

#[inline(always)]
pub fn index_info_from_raw<'a>(raw: *const RawIndexInfo) -> Option<IndexInfo<'a>> {
    if raw.is_null() {
        None
    } else {
        Some(IndexInfo::from_raw(raw))
    }
}

#[inline(always)]
pub fn conditional_root_from_raw<'a>(
    raw: *const RawConditionalRoot,
) -> Option<ConditionalRoot<'a>> {
    if raw.is_null() {
        None
    } else {
        Some(ConditionalRoot::from_raw(raw))
    }
}

#[inline(always)]
pub fn visit_node<'a, F>(f: &mut F, node: *const RawNode) -> bool
where
    F: FnMut(Node<'a>) -> bool,
{
    if let Some(node) = node_from_raw(node) {
        f(node)
    } else {
        false
    }
}

#[inline(always)]
pub fn visit_node_list<'a, F>(f: &mut F, node_list: *const RawNodeList) -> bool
where
    F: FnMut(Node<'a>) -> bool,
{
    if node_list.is_null() {
        return false;
    }

    let node_list = NodeList::from_raw(node_list);
    for idx in 0..node_list.len() {
        if f(node_list.get(idx).unwrap()) {
            return true;
        }
    }

    false
}
