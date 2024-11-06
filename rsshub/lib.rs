use anyhow::{anyhow, Result};
use biome_js_parser::{parse, JsParserOptions};
use biome_js_syntax::{
    AnyJsArrayBindingPatternElement, AnyJsBinding, AnyJsBindingPattern, AnyJsCombinedSpecifier,
    AnyJsDeclarationClause, AnyJsExportClause, AnyJsExportDefaultDeclaration,
    AnyJsExportNamedSpecifier, AnyJsImportClause, AnyJsNamedImportSpecifier,
    AnyJsObjectBindingPatternMember, JsExport, JsFileSource, JsFormalParameter, JsImport,
    JsNamedImportSpecifiers, JsSyntaxKind, TsAsExpression, TsNonNullAssertionExpression,
    TsSatisfiesExpression, TsTypeAssertionExpression,
};
use biome_rowan::{AstNode, Direction, NodeOrToken, SyntaxResult, TokenText, WalkEvent};
use itertools::Itertools;
use rustc_hash::FxHashSet;
use std::{
    ffi::{c_char, c_int, CString},
    fmt::Write,
    slice, str,
};

#[no_mangle]
pub extern "C" fn transform(
    src: *const u8,
    src_len: c_int,
    path: *const u8,
    path_len: c_int,
) -> *mut c_char {
    let src = match transform_impl(src, src_len, path, path_len) {
        Ok(mut src) => {
            src.push('1');
            src
        }
        Err(error) => {
            let mut s = error.to_string();
            s.push('2');
            s
        }
    }
    .into_bytes();
    unsafe { CString::from_vec_unchecked(src) }.into_raw()
}

#[allow(clippy::missing_safety_doc)]
#[no_mangle]
pub unsafe extern "C" fn retrieve(s: *mut c_char) {
    _ = CString::from_raw(s)
}

fn transform_impl(
    src: *const u8,
    src_len: c_int,
    path: *const u8,
    path_len: c_int,
) -> Result<String> {
    let src = unsafe { str::from_utf8_unchecked(slice::from_raw_parts(src, src_len.try_into()?)) };
    let path =
        unsafe { str::from_utf8_unchecked(slice::from_raw_parts(path, path_len.try_into()?)) };
    let parsed = parse(src, JsFileSource::ts(), JsParserOptions::default());
    let diagnostics = parsed.diagnostics();
    if !diagnostics.is_empty() {
        return Err(anyhow!(
            "{}",
            diagnostics
                .iter()
                .map(|diag| format!("{diag:?}"))
                .join("\n")
        ));
    }

    let mut it = parsed.syntax().preorder_with_tokens(Direction::Next);
    let mut buf = String::from("(function(exports,require,module){");
    let mut blanks = FxHashSet::default();
    let mut skip = FxHashSet::default();
    let mut exports = Vec::new();

    // TODO: for await
    // TODO: ??=
    // TODO: ||=
    // TODO: https://github.com/bloomberg/ts-blank-space#arrow-function-type-annotations-that-introduce-a-new-line
    // TODO: enum
    // TODO: export { _ as default } from
    while let Some(event) = it.next() {
        use JsSyntaxKind::*;
        match event {
            WalkEvent::Enter(NodeOrToken::Node(syntax)) => match syntax.kind() {
                TS_TYPE_ARGUMENTS
                | TS_TYPE_PARAMETERS
                | TS_TYPE_ANNOTATION
                | TS_RETURN_TYPE_ANNOTATION
                | TS_DEFINITE_PROPERTY_ANNOTATION
                | TS_OPTIONAL_PROPERTY_ANNOTATION
                | TS_DECLARE_FUNCTION_DECLARATION
                | TS_INTERFACE_DECLARATION
                | TS_TYPE_ALIAS_DECLARATION => {
                    blank(&mut buf, syntax.text().chars());
                    it.skip_subtree();
                }
                TS_THIS_PARAMETER => unimplemented!(),
                JS_FORMAL_PARAMETER => {
                    let param = JsFormalParameter::cast(syntax).unwrap();
                    if let Some(token) = param.question_mark_token() {
                        blanks.insert(token.key());
                    }
                }
                TS_NON_NULL_ASSERTION_EXPRESSION => {
                    let expr = TsNonNullAssertionExpression::cast(syntax).unwrap();
                    blanks.insert(expr.excl_token()?.key());
                }
                TS_AS_EXPRESSION => {
                    let expr = TsAsExpression::cast(syntax).unwrap();
                    blanks.insert(expr.as_token()?.key());
                    blanks.extend(
                        expr.ty()?
                            .syntax()
                            .descendants_tokens(Direction::Next)
                            .map(|t| t.key()),
                    );
                }
                TS_TYPE_ASSERTION_EXPRESSION => {
                    let expr = TsTypeAssertionExpression::cast(syntax).unwrap();
                    blanks.insert(expr.l_angle_token()?.key());
                    blanks.insert(expr.r_angle_token()?.key());
                    blanks.extend(
                        expr.ty()?
                            .syntax()
                            .descendants_tokens(Direction::Next)
                            .map(|t| t.key()),
                    );
                }
                TS_SATISFIES_EXPRESSION => {
                    let expr = TsSatisfiesExpression::cast(syntax).unwrap();
                    blanks.insert(expr.satisfies_token()?.key());
                    blanks.extend(
                        expr.ty()?
                            .syntax()
                            .descendants_tokens(Direction::Next)
                            .map(|t| t.key()),
                    );
                }
                JS_IMPORT_CALL_EXPRESSION => {
                    if let Some(trivia) = syntax.first_leading_trivia() {
                        buf.push_str(trivia.text());
                    }
                    buf.push('{');
                    for _ in 0..usize::from(syntax.text_trimmed_range().len()) - 2 {
                        buf.push(' ');
                    }
                    buf.push('}');
                    if let Some(trivia) = syntax.last_trailing_trivia() {
                        buf.push_str(trivia.text());
                    }
                    it.skip_subtree();
                }
                JS_IMPORT_META_EXPRESSION => {
                    _ = write!(buf, "({{ url: '{}' }})", path);
                    it.skip_subtree();
                }
                JS_IMPORT => {
                    if let Some(trivia) = syntax.first_leading_trivia() {
                        buf.push_str(trivia.text());
                    }
                    let import = JsImport::cast(syntax).unwrap();
                    match import.import_clause()? {
                        AnyJsImportClause::JsImportBareClause(clause) => {
                            _ = write!(
                                buf,
                                "require('{}');",
                                clause.source()?.inner_string_text()?
                            );
                        }
                        AnyJsImportClause::JsImportCombinedClause(clause) => {
                            let name = name_token(clause.default_specifier()?.local_name()?)?;
                            let source = clause.source()?.inner_string_text()?;
                            match clause.specifier()? {
                                AnyJsCombinedSpecifier::JsNamedImportSpecifiers(specifiers) => {
                                    let mut imports = collect_imports(specifiers)?;
                                    imports.push(format!("default: {name}"));
                                    _ = write!(
                                        buf,
                                        "const {{ {} }} = require('{source}');",
                                        imports.join(", "),
                                    );
                                }
                                AnyJsCombinedSpecifier::JsNamespaceImportSpecifier(specifier) => {
                                    let local_name = name_token(specifier.local_name()?)?;
                                    _ = write!(buf, "const {local_name} = require('{source}');");
                                    _ = write!(buf, " const {name} = {local_name}.default;");
                                }
                            };
                        }
                        AnyJsImportClause::JsImportDefaultClause(clause)
                            if clause.type_token().is_none() =>
                        {
                            _ = write!(
                                buf,
                                "const {} = require('{}').default;",
                                name_token(clause.default_specifier()?.local_name()?)?,
                                clause.source()?.inner_string_text()?
                            );
                        }
                        AnyJsImportClause::JsImportNamedClause(clause)
                            if clause.type_token().is_none() =>
                        {
                            _ = write!(
                                buf,
                                "const {{ {} }} = require('{}');",
                                collect_imports(clause.named_specifiers()?)?.join(", "),
                                clause.source()?.inner_string_text()?
                            );
                        }
                        AnyJsImportClause::JsImportNamespaceClause(clause)
                            if clause.type_token().is_none() =>
                        {
                            _ = write!(
                                buf,
                                "const {} = require('{}');",
                                name_token(clause.namespace_specifier()?.local_name()?)?,
                                clause.source()?.inner_string_text()?
                            );
                        }
                        _ => (),
                    }
                    for _ in 0..import
                        .trim_leading_trivia()
                        .map(|import| newlines(import.syntax().text().chars()))
                        .unwrap_or(0)
                    {
                        buf.push('\n');
                    }
                    it.skip_subtree();
                }
                JS_EXPORT => {
                    let export = JsExport::cast(syntax).unwrap();
                    match export.export_clause()? {
                        AnyJsExportClause::AnyJsDeclarationClause(clause) => {
                            blanks.insert(export.export_token()?.key());
                            match clause {
                                AnyJsDeclarationClause::JsClassDeclaration(decl) => {
                                    exports.push(name_token(decl.id()?)?);
                                }
                                AnyJsDeclarationClause::JsFunctionDeclaration(decl) => {
                                    exports.push(name_token(decl.id()?)?);
                                }
                                AnyJsDeclarationClause::JsVariableDeclarationClause(decl) => {
                                    for decl in decl.declaration()?.declarators() {
                                        collect_exports(&mut exports, decl?.id()?)?;
                                    }
                                }
                                AnyJsDeclarationClause::TsDeclareFunctionDeclaration(_)
                                | AnyJsDeclarationClause::TsInterfaceDeclaration(_)
                                | AnyJsDeclarationClause::TsTypeAliasDeclaration(_) => (),
                                AnyJsDeclarationClause::TsEnumDeclaration(_) => {
                                    unimplemented!()
                                }
                                AnyJsDeclarationClause::TsExternalModuleDeclaration(_)
                                | AnyJsDeclarationClause::TsGlobalDeclaration(_)
                                | AnyJsDeclarationClause::TsImportEqualsDeclaration(_)
                                | AnyJsDeclarationClause::TsModuleDeclaration(_) => {
                                    unreachable!()
                                }
                            }
                        }
                        AnyJsExportClause::JsExportDefaultDeclarationClause(clause) => {
                            let export_token = export.export_token()?;
                            skip.insert(export_token.key());
                            skip.insert(clause.default_token()?.key());
                            use AnyJsExportDefaultDeclaration::*;
                            match clause.declaration()? {
                                JsClassExportDefaultDeclaration(_)
                                | JsFunctionExportDefaultDeclaration(_) => {
                                    buf.push_str(export_token.leading_trivia().text());
                                    buf.push_str("module.exports.default = ");
                                }
                                TsDeclareFunctionExportDefaultDeclaration(_)
                                | TsInterfaceDeclaration(_) => (),
                            }
                        }
                        AnyJsExportClause::JsExportDefaultExpressionClause(clause) => {
                            let export_token = export.export_token()?;
                            buf.push_str(export_token.leading_trivia().text());
                            skip.insert(export_token.key());
                            skip.insert(clause.default_token()?.key());
                            buf.push_str("module.exports.default = ");
                        }
                        AnyJsExportClause::JsExportNamedClause(clause) => {
                            if clause.type_token().is_some() {
                                blank(&mut buf, export.syntax().text().chars());
                                it.skip_subtree();
                            } else {
                                let mut exports = Vec::new();
                                for specifier in clause.specifiers() {
                                    use AnyJsExportNamedSpecifier::*;
                                    match specifier? {
                                        JsExportNamedShorthandSpecifier(specifier)
                                            if specifier.type_token().is_none() =>
                                        {
                                            exports.push(specifier.name()?.name()?.to_string());
                                        }
                                        JsExportNamedSpecifier(specifier)
                                            if specifier.type_token().is_none() =>
                                        {
                                            exports.push(format!(
                                                "{}: {}",
                                                specifier.exported_name()?.value()?.text_trimmed(),
                                                specifier.local_name()?.name()?
                                            ));
                                        }
                                        _ => (),
                                    }
                                }
                                if let Some(trivia) = export.syntax().first_leading_trivia() {
                                    buf.push_str(trivia.text());
                                }
                                _ = write!(
                                    buf,
                                    "Object.assign(module.exports, {{ {} }});",
                                    exports.join(", ")
                                );
                                if let Some(export) = export.trim_leading_trivia() {
                                    for _ in 0..newlines(export.syntax().text().chars()) {
                                        buf.push('\n');
                                    }
                                }
                                it.skip_subtree();
                            }
                        }
                        AnyJsExportClause::JsExportNamedFromClause(clause) => {
                            if clause.type_token().is_some() {
                                blank(&mut buf, export.syntax().text().chars());
                                it.skip_subtree();
                            } else {
                                let mut exports = Vec::new();
                                for specifier in clause.specifiers() {
                                    let specifier = specifier?;
                                    if specifier.type_token().is_none() {
                                        let source_name =
                                            specifier.source_name()?.inner_string_text()?;
                                        let export_as = match specifier.export_as() {
                                            Some(clause) => clause
                                                .exported_name()?
                                                .value()?
                                                .token_text_trimmed(),
                                            None => source_name.clone(),
                                        };
                                        exports.push(format!(
                                            "{export_as}: require('{}').{source_name}",
                                            clause.source()?.inner_string_text()?,
                                        ))
                                    }
                                }
                                if let Some(trivia) = export.syntax().first_leading_trivia() {
                                    buf.push_str(trivia.text());
                                }
                                _ = write!(
                                    buf,
                                    "Object.assign(module.exports, {{ {} }});",
                                    exports.join(", ")
                                );
                                if let Some(export) = export.trim_leading_trivia() {
                                    for _ in 0..newlines(export.syntax().text().chars()) {
                                        buf.push('\n');
                                    }
                                }
                                it.skip_subtree();
                            }
                        }
                        AnyJsExportClause::JsExportFromClause(_) => unreachable!(),
                        AnyJsExportClause::TsExportAsNamespaceClause(_) => unreachable!(),
                        AnyJsExportClause::TsExportAssignmentClause(_) => unreachable!(),
                        AnyJsExportClause::TsExportDeclareClause(_) => unreachable!(),
                    }
                }
                _ => (),
            },
            WalkEvent::Enter(NodeOrToken::Token(token)) => {
                let key = token.key();
                if blanks.remove(&key) {
                    blank(&mut buf, token.text().chars());
                    it.skip_subtree();
                } else if skip.remove(&key) {
                    it.skip_subtree();
                } else {
                    buf.push_str(token.text());
                }
            }
            _ => (),
        };
    }

    if !exports.is_empty() {
        buf.push('\n');
        for export in exports {
            _ = writeln!(buf, "module.exports.{export} = {export};");
        }
    }
    buf.push_str("\n})");
    Ok(buf)
}

fn collect_exports(exports: &mut Vec<TokenText>, pat: AnyJsBindingPattern) -> SyntaxResult<()> {
    match pat {
        AnyJsBindingPattern::AnyJsBinding(binding) => {
            exports.push(name_token(binding)?);
        }
        AnyJsBindingPattern::JsArrayBindingPattern(pat) => {
            for elem in pat.elements() {
                match elem? {
                    AnyJsArrayBindingPatternElement::JsArrayBindingPatternElement(elem) => {
                        collect_exports(exports, elem.pattern()?)?
                    }
                    AnyJsArrayBindingPatternElement::JsArrayBindingPatternRestElement(elem) => {
                        collect_exports(exports, elem.pattern()?)?
                    }
                    AnyJsArrayBindingPatternElement::JsArrayHole(_) => (),
                }
            }
        }
        AnyJsBindingPattern::JsObjectBindingPattern(pat) => {
            for prop in pat.properties() {
                use AnyJsObjectBindingPatternMember::*;
                match prop? {
                    JsBogusBinding(_) => unreachable!(),
                    JsObjectBindingPatternProperty(prop) => {
                        collect_exports(exports, prop.pattern()?)?
                    }
                    JsObjectBindingPatternRest(pat) => exports.push(name_token(pat.binding()?)?),
                    JsObjectBindingPatternShorthandProperty(prop) => {
                        exports.push(name_token(prop.identifier()?)?)
                    }
                }
            }
        }
    }
    Ok(())
}

fn collect_imports(specifiers: JsNamedImportSpecifiers) -> SyntaxResult<Vec<String>> {
    let mut imports = Vec::new();
    for specifier in specifiers.specifiers() {
        match specifier? {
            AnyJsNamedImportSpecifier::JsBogusNamedImportSpecifier(_) => unreachable!(),
            AnyJsNamedImportSpecifier::JsNamedImportSpecifier(specifier)
                if specifier.type_token().is_none() =>
            {
                imports.push(format!(
                    "{}: {}",
                    specifier.name()?.value()?.token_text_trimmed(),
                    name_token(specifier.local_name()?)?
                ))
            }
            AnyJsNamedImportSpecifier::JsShorthandNamedImportSpecifier(specifier)
                if specifier.type_token().is_none() =>
            {
                imports.push(name_token(specifier.local_name()?)?.to_string())
            }
            _ => (),
        }
    }
    Ok(imports)
}

fn blank(buf: &mut String, chars: impl Iterator<Item = char>) {
    for char in chars {
        buf.push(if char.is_ascii_whitespace() {
            char
        } else {
            ' '
        });
    }
}

fn name_token(binding: AnyJsBinding) -> SyntaxResult<TokenText> {
    Ok(binding
        .as_js_identifier_binding()
        .unwrap()
        .name_token()?
        .token_text_trimmed())
}

fn newlines(chars: impl Iterator<Item = char>) -> usize {
    chars.filter(|c| *c == '\n').count()
}
