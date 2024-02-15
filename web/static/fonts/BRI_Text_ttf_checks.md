## Fontbakery report

Fontbakery version: 0.8.8

<details><summary><b>[10] Family checks</b></summary><div><details><summary>🍞 <b>PASS:</b> Checking all files are in the same directory. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/single_directory">com.google.fonts/check/family/single_directory</a>)</summary><div>

>
>If the set of font files passed in the command line is not all in the same directory, then we warn the user since the tool will interpret the set of files as belonging to a single family (and it is unlikely that the user would store the files from a single family spreaded in several separate directories).
>
>
* 🍞 **PASS** All files are in the same directory.
</div></details><details><summary>🍞 <b>PASS:</b> Each font in a family must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/vertical_metrics">com.google.fonts/check/family/vertical_metrics</a>)</summary><div>

>
>We want all fonts within a family to have the same vertical metrics so their line spacing is consistent across the family.
>
>
* 🍞 **PASS** Vertical metrics are the same across the family.
</div></details><details><summary>🍞 <b>PASS:</b> Fonts have equal unicode encodings? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/family/equal_unicode_encodings">com.google.fonts/check/family/equal_unicode_encodings</a>)</summary><div>


* 🍞 **PASS** Fonts have equal unicode encodings.
</div></details><details><summary>🍞 <b>PASS:</b> Make sure all font files have the same version value. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/family/equal_font_versions">com.google.fonts/check/family/equal_font_versions</a>)</summary><div>


* 🍞 **PASS** All font files have the same version.
</div></details><details><summary>🍞 <b>PASS:</b> Fonts have consistent PANOSE proportion? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/family/panose_proportion">com.google.fonts/check/family/panose_proportion</a>)</summary><div>


* 🍞 **PASS** Fonts have consistent PANOSE proportion.
</div></details><details><summary>🍞 <b>PASS:</b> Fonts have consistent PANOSE family type? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/family/panose_familytype">com.google.fonts/check/family/panose_familytype</a>)</summary><div>


* 🍞 **PASS** Fonts have consistent PANOSE family type.
</div></details><details><summary>🍞 <b>PASS:</b> Check that OS/2.fsSelection bold & italic settings are unique for each NameID1 (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/family/bold_italic_unique_for_nameid1">com.adobe.fonts/check/family/bold_italic_unique_for_nameid1</a>)</summary><div>

>
>Per the OpenType spec: name ID 1 'is used in combination with Font Subfamily name (name ID 2), and should be shared among at most four fonts that differ only in weight or style...
>
>This four-way distinction should also be reflected in the OS/2.fsSelection field, using bits 0 and 5.
>
>
* 🍞 **PASS** The OS/2.fsSelection bold & italic settings were unique within each compatible family group.
</div></details><details><summary>🍞 <b>PASS:</b> Fonts have consistent underline thickness? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/family/underline_thickness">com.google.fonts/check/family/underline_thickness</a>)</summary><div>

>
>Dave C Lemon (Adobe Type Team) recommends setting the underline thickness to be consistent across the family.
>
>If thicknesses are not family consistent, words set on the same line which have different styles look strange.
>
>See also:
>https://twitter.com/typenerd1/status/690361887926697986
>
>
* 🍞 **PASS** Fonts have consistent underline thickness.
</div></details><details><summary>🍞 <b>PASS:</b> Verify that each group of fonts with the same nameID 1 has maximum of 4 fonts (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/family/max_4_fonts_per_family_name">com.adobe.fonts/check/family/max_4_fonts_per_family_name</a>)</summary><div>

>
>Per the OpenType spec:
>'The Font Family name [...] should be shared among at most four fonts that differ only in weight or style [...]'
>
>
* 🍞 **PASS** There were no more than 4 fonts per family name.
</div></details><details><summary>💤 <b>SKIP:</b> Do we have the latest version of FontBakery installed? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/fontbakery_version">com.google.fonts/check/fontbakery_version</a>)</summary><div>

>
>Running old versions of FontBakery can lead to a poor report which may include false WARNs and FAILs due do bugs, as well as outdated quality assurance criteria.
>
>Older versions will also not report problems that are detected by new checks added to the tool in more recent updates.
>
>
* 💤 **SKIP** No applicable arguments
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-Bold.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* atilde (U+00E3): X=148.5,Y=702.0 (should be at cap-height 700?)
	* aring (U+00E5): X=394.0,Y=699.0 (should be at cap-height 700?)
	* aring (U+00E5): X=192.0,Y=699.0 (should be at cap-height 700?)
	* aring (U+00E5): X=256.0,Y=699.0 (should be at cap-height 700?)
	* aring (U+00E5): X=328.0,Y=699.0 (should be at cap-height 700?)
	* ntilde (U+00F1): X=164.5,Y=702.0 (should be at cap-height 700?)
	* otilde (U+00F5): X=162.5,Y=702.0 (should be at cap-height 700?)
	* florin (U+0192): X=190.0,Y=701.0 (should be at cap-height 700?)
	* ring (U+02DA): X=255.0,Y=699.0 (should be at cap-height 700?)
	* ring (U+02DA): X=53.0,Y=699.0 (should be at cap-height 700?) and 10 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* at (U+0040) contains a short segment L<<720.0,-19.0>--<702.0,-19.0>>
	* at (U+0040) contains a short segment B<<720.0,147.0>-<699.0,147.0>-<687.5,162.0>>
	* at (U+0040) contains a short segment L<<562.0,475.0>--<556.0,441.0>>
	* G (U+0047) contains a short segment L<<532.0,253.0>--<532.0,246.0>>
	* K (U+004B) contains a short segment L<<665.0,0.0>--<665.0,18.0>>
	* K (U+004B) contains a short segment L<<650.0,682.0>--<650.0,700.0>>
	* Q (U+0051) contains a short segment B<<384.0,-12.0>-<389.0,-12.0>-<395.0,-12.0>>
	* V (U+0056) contains a short segment L<<675.0,682.0>--<675.0,700.0>>
	* V (U+0056) contains a short segment L<<351.0,180.0>--<341.0,180.0>>
	* V (U+0056) contains a short segment L<<16.0,700.0>--<16.0,682.0>> and 36 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* 🍞 **PASS** No jaggy segments found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 🍞 **PASS** No semi-horizontal/semi-vertical lines found.
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-BoldItalic.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* dollar (U+0024): X=465.0,Y=701.0 (should be at cap-height 700?)
	* b (U+0062): X=327.5,Y=531.5 (should be at x-height 530?)
	* p (U+0070): X=319.5,Y=528.5 (should be at x-height 530?)
	* atilde (U+00E3): X=218.0,Y=702.0 (should be at cap-height 700?)
	* ntilde (U+00F1): X=246.0,Y=702.0 (should be at cap-height 700?)
	* otilde (U+00F5): X=240.0,Y=702.0 (should be at cap-height 700?)
	* florin (U+0192): X=268.0,Y=701.0 (should be at cap-height 700?)
	* tilde (U+02DC): X=114.0,Y=702.0 (should be at cap-height 700?)
	* tildecomb (U+0303): X=114.0,Y=702.0 (should be at cap-height 700?)
	* uni208E (U+208E): X=25.5,Y=0.5 (should be at baseline 0?)
	* emptyset (U+2205): X=282.5,Y=1.5 (should be at baseline 0?) and emptyset (U+2205): X=563.5,Y=698.0 (should be at cap-height 700?) [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* at (U+0040) contains a short segment L<<677.0,-36.0>--<659.0,-35.0>>
	* at (U+0040) contains a short segment B<<711.0,151.0>-<687.0,151.0>-<679.0,164.0>>
	* at (U+0040) contains a short segment L<<605.0,472.0>--<594.0,439.0>>
	* G (U+0047) contains a short segment L<<540.0,253.0>--<539.0,246.0>>
	* K (U+004B) contains a short segment L<<619.0,0.0>--<622.0,18.0>>
	* K (U+004B) contains a short segment L<<723.0,682.0>--<727.0,700.0>>
	* V (U+0056) contains a short segment L<<741.0,682.0>--<745.0,700.0>>
	* V (U+0056) contains a short segment L<<322.0,180.0>--<312.0,180.0>>
	* V (U+0056) contains a short segment L<<86.0,700.0>--<82.0,682.0>>
	* W (U+0057) contains a short segment L<<519.0,428.0>--<529.0,428.0>> and 36 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>⚠ <b>WARN:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* ⚠ **WARN** The following glyphs have jaggy segments:
	* dollar (U+0024): B<<112.0,44.5>-<156.0,14.0>-<214.0,0.0>>/L<<214.0,0.0>--<202.0,0.0>> = 13.570434385161475 [code: found-jaggy-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_not_italic
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-Heavy.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* ampersand (U+0026): X=357.5,Y=2.0 (should be at baseline 0?)
	* cent (U+00A2): X=225.0,Y=-1.0 (should be at baseline 0?)
	* breve (U+02D8): X=242.0,Y=701.0 (should be at cap-height 700?)
	* breve (U+02D8): X=167.5,Y=701.0 (should be at cap-height 700?)
	* uni0306 (U+0306): X=242.0,Y=701.0 (should be at cap-height 700?)
	* uni0306 (U+0306): X=167.5,Y=701.0 (should be at cap-height 700?)
	* emptyset (U+2205): X=303.5,Y=1.0 (should be at baseline 0?) and emptyset (U+2205): X=505.0,Y=699.5 (should be at cap-height 700?) [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* question (U+003F) contains a short segment L<<355.0,250.0>--<355.0,261.0>>
	* at (U+0040) contains a short segment L<<715.0,-4.0>--<695.0,-4.0>>
	* at (U+0040) contains a short segment B<<712.0,166.0>-<693.0,166.0>-<684.0,180.0>>
	* at (U+0040) contains a short segment B<<684.0,180.0>-<675.0,194.0>-<675.0,213.0>>
	* at (U+0040) contains a short segment L<<544.0,471.0>--<539.0,445.0>>
	* G (U+0047) contains a short segment L<<512.0,239.0>--<512.0,235.0>>
	* K (U+004B) contains a short segment L<<676.0,0.0>--<676.0,20.0>>
	* K (U+004B) contains a short segment L<<664.0,680.0>--<664.0,700.0>>
	* V (U+0056) contains a short segment L<<356.0,202.0>--<346.0,202.0>>
	* W (U+0057) contains a short segment L<<499.0,389.0>--<509.0,389.0>> and 35 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* 🍞 **PASS** No jaggy segments found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 🍞 **PASS** No semi-horizontal/semi-vertical lines found.
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-HeavyItalic.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* ampersand (U+0026): X=316.5,Y=1.5 (should be at baseline 0?)
	* d (U+0064): X=384.5,Y=530.0 (should be at x-height 532?)
	* p (U+0070): X=334.0,Y=532.5 (should be at x-height 532?)
	* breve (U+02D8): X=319.5,Y=701.0 (should be at cap-height 700?)
	* breve (U+02D8): X=245.0,Y=701.0 (should be at cap-height 700?)
	* uni0306 (U+0306): X=319.5,Y=701.0 (should be at cap-height 700?)
	* uni0306 (U+0306): X=245.0,Y=701.0 (should be at cap-height 700?)
	* uni208E (U+208E): X=22.5,Y=2.0 (should be at baseline 0?)
	* emptyset (U+2205): X=280.5,Y=1.0 (should be at baseline 0?) and emptyset (U+2205): X=555.5,Y=699.0 (should be at cap-height 700?) [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* question (U+003F) contains a short segment L<<353.0,250.0>--<355.0,261.0>>
	* at (U+0040) contains a short segment L<<675.0,-21.0>--<655.0,-20.0>>
	* at (U+0040) contains a short segment B<<734.5,186.0>-<723.0,171.0>-<705.0,171.0>>
	* at (U+0040) contains a short segment B<<705.0,171.0>-<686.0,171.0>-<679.5,183.0>>
	* at (U+0040) contains a short segment L<<587.0,469.0>--<578.0,444.0>>
	* G (U+0047) contains a short segment L<<517.0,239.0>--<517.0,235.0>>
	* K (U+004B) contains a short segment L<<629.0,0.0>--<633.0,20.0>>
	* K (U+004B) contains a short segment L<<737.0,680.0>--<741.0,700.0>>
	* Q (U+0051) contains a short segment B<<367.0,-12.0>-<374.0,-12.0>-<381.0,-12.0>>
	* V (U+0056) contains a short segment L<<327.0,202.0>--<317.0,202.0>> and 37 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>⚠ <b>WARN:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* ⚠ **WARN** The following glyphs have jaggy segments:
	* dollar (U+0024): B<<108.5,44.5>-<153.0,14.0>-<212.0,0.0>>/L<<212.0,0.0>--<191.0,0.0>> = 13.348727113287385 and dollar (U+0024): B<<566.0,656.5>-<525.0,686.0>-<469.0,700.0>>/L<<469.0,700.0>--<474.0,700.0>> = 14.036243467926484 [code: found-jaggy-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_not_italic
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-Italic.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* percent (U+0025): X=225.0,Y=-1.0 (should be at baseline 0?)
	* percent (U+0025): X=779.0,Y=699.0 (should be at cap-height 700?)
	* less (U+003C): X=533.0,Y=-2.0 (should be at baseline 0?)
	* greater (U+003E): X=75.0,Y=-2.0 (should be at baseline 0?)
	* Q (U+0051): X=461.0,Y=-1.0 (should be at baseline 0?)
	* g (U+0067): X=418.0,Y=-2.0 (should be at baseline 0?)
	* questiondown (U+00BF): X=447.0,Y=-1.0 (should be at baseline 0?)
	* questiondown (U+00BF): X=368.0,Y=-1.0 (should be at baseline 0?)
	* aring (U+00E5): X=382.0,Y=699.0 (should be at cap-height 700?)
	* ring (U+02DA): X=271.0,Y=699.0 (should be at cap-height 700?) and 5 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* dollar (U+0024) contains a short segment B<<312.0,-12.0>-<317.0,-12.0>-<321.0,-12.0>>
	* ampersand (U+0026) contains a short segment L<<568.0,0.0>--<569.0,12.0>>
	* ampersand (U+0026) contains a short segment L<<622.0,319.0>--<623.0,331.0>>
	* parenleft (U+0028) contains a short segment L<<215.0,-170.0>--<217.0,-158.0>>
	* parenright (U+0029) contains a short segment L<<105.0,690.0>--<103.0,678.0>>
	* seven (U+0037) contains a short segment L<<59.0,12.0>--<57.0,0.0>>
	* at (U+0040) contains a short segment L<<685.0,-84.0>--<673.0,-83.0>>
	* A (U+0041) contains a short segment L<<618.0,0.0>--<620.0,12.0>>
	* A (U+0041) contains a short segment L<<-20.0,12.0>--<-22.0,0.0>>
	* G (U+0047) contains a short segment L<<610.0,297.0>--<608.0,281.0>> and 75 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_not_italic
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* 🍞 **PASS** No jaggy segments found.
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-Light.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* i (U+0069): X=146.0,Y=698.0 (should be at cap-height 700?)
	* i (U+0069): X=76.0,Y=698.0 (should be at cap-height 700?)
	* j (U+006A): X=158.0,Y=698.0 (should be at cap-height 700?)
	* j (U+006A): X=88.0,Y=698.0 (should be at cap-height 700?)
	* section (U+00A7): X=158.5,Y=1.0 (should be at baseline 0?)
	* Agrave (U+00C0): X=295.0,Y=918.0 (should be at ascender 920?)
	* Agrave (U+00C0): X=237.0,Y=918.0 (should be at ascender 920?)
	* Aacute (U+00C1): X=461.0,Y=918.0 (should be at ascender 920?)
	* Aacute (U+00C1): X=403.0,Y=918.0 (should be at ascender 920?)
	* Acircumflex (U+00C2): X=381.0,Y=918.0 (should be at ascender 920?) and 49 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* percent (U+0025) contains a short segment L<<664.0,690.0>--<664.0,700.0>>
	* percent (U+0025) contains a short segment L<<174.0,10.0>--<174.0,0.0>>
	* ampersand (U+0026) contains a short segment L<<604.0,0.0>--<604.0,10.0>>
	* ampersand (U+0026) contains a short segment L<<602.0,315.0>--<602.0,325.0>>
	* parenleft (U+0028) contains a short segment L<<280.0,-170.0>--<280.0,-160.0>>
	* parenright (U+0029) contains a short segment L<<35.0,690.0>--<35.0,680.0>>
	* seven (U+0037) contains a short segment L<<128.0,10.0>--<128.0,0.0>>
	* at (U+0040) contains a short segment L<<744.0,-82.0>--<734.0,-82.0>>
	* A (U+0041) contains a short segment L<<664.0,0.0>--<664.0,10.0>>
	* A (U+0041) contains a short segment L<<34.0,10.0>--<34.0,0.0>> and 80 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* 🍞 **PASS** No jaggy segments found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 🍞 **PASS** No semi-horizontal/semi-vertical lines found.
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-LightItalic.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* Q (U+0051): X=462.0,Y=-2.0 (should be at baseline 0?)
	* i (U+0069): X=227.0,Y=698.0 (should be at cap-height 700?)
	* i (U+0069): X=157.0,Y=698.0 (should be at cap-height 700?)
	* j (U+006A): X=229.0,Y=698.0 (should be at cap-height 700?)
	* j (U+006A): X=159.0,Y=698.0 (should be at cap-height 700?)
	* m (U+006D): X=618.0,Y=523.0 (should be at x-height 522?)
	* section (U+00A7): X=425.0,Y=2.0 (should be at baseline 0?)
	* section (U+00A7): X=125.0,Y=1.0 (should be at baseline 0?)
	* Agrave (U+00C0): X=426.0,Y=918.0 (should be at ascender 920?)
	* Agrave (U+00C0): X=368.0,Y=918.0 (should be at ascender 920?) and 55 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* dollar (U+0024) contains a short segment B<<308.0,-12.0>-<314.0,-12.0>-<320.0,-12.0>>
	* percent (U+0025) contains a short segment L<<759.0,690.0>--<760.0,700.0>>
	* percent (U+0025) contains a short segment L<<145.0,10.0>--<144.0,0.0>>
	* ampersand (U+0026) contains a short segment L<<560.0,0.0>--<561.0,10.0>>
	* ampersand (U+0026) contains a short segment L<<614.0,315.0>--<615.0,325.0>>
	* parenleft (U+0028) contains a short segment L<<203.0,-170.0>--<205.0,-160.0>>
	* parenright (U+0029) contains a short segment L<<111.0,690.0>--<109.0,680.0>>
	* seven (U+0037) contains a short segment L<<78.0,10.0>--<76.0,0.0>>
	* at (U+0040) contains a short segment L<<691.0,-101.0>--<681.0,-100.0>>
	* A (U+0041) contains a short segment L<<618.0,0.0>--<620.0,10.0>> and 84 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_not_italic
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* 🍞 **PASS** No jaggy segments found.
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-Medium.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* section (U+00A7): X=270.0,Y=1.0 (should be at baseline 0?) and questiondown (U+00BF): X=168.0,Y=-1.0 (should be at baseline 0?) [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* ampersand (U+0026) contains a short segment L<<623.0,0.0>--<623.0,13.0>>
	* ampersand (U+0026) contains a short segment L<<620.0,322.0>--<620.0,335.0>>
	* seven (U+0037) contains a short segment L<<103.0,13.0>--<103.0,0.0>>
	* at (U+0040) contains a short segment L<<732.0,-55.0>--<719.0,-55.0>>
	* A (U+0041) contains a short segment L<<668.0,0.0>--<668.0,13.0>>
	* A (U+0041) contains a short segment L<<20.0,13.0>--<20.0,0.0>>
	* G (U+0047) contains a short segment L<<582.0,287.0>--<582.0,273.0>>
	* K (U+004B) contains a short segment L<<640.0,0.0>--<640.0,13.0>>
	* K (U+004B) contains a short segment L<<615.0,687.0>--<615.0,700.0>>
	* M (U+004D) contains a short segment L<<429.0,133.0>--<457.0,133.0>> and 65 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* 🍞 **PASS** No jaggy segments found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 🍞 **PASS** No semi-horizontal/semi-vertical lines found.
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-MediumItalic.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* dollar (U+0024): X=229.0,Y=-2.0 (should be at baseline 0?)
	* percent (U+0025): X=250.0,Y=-1.0 (should be at baseline 0?)
	* percent (U+0025): X=804.0,Y=699.0 (should be at cap-height 700?)
	* g (U+0067): X=408.0,Y=-2.0 (should be at baseline 0?)
	* section (U+00A7): X=241.0,Y=1.0 (should be at baseline 0?)
	* ogonek (U+02DB): X=212.5,Y=-0.5 (should be at baseline 0?)
	* uni0328 (U+0328): X=212.5,Y=-0.5 (should be at baseline 0?)
	* perthousand (U+2030): X=250.0,Y=-1.0 (should be at baseline 0?) and perthousand (U+2030): X=804.0,Y=699.0 (should be at cap-height 700?) [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* dollar (U+0024) contains a short segment L<<319.0,-12.0>--<321.0,-12.0>>
	* ampersand (U+0026) contains a short segment L<<578.0,0.0>--<579.0,13.0>>
	* ampersand (U+0026) contains a short segment L<<632.0,322.0>--<633.0,335.0>>
	* seven (U+0037) contains a short segment L<<49.0,13.0>--<47.0,0.0>>
	* at (U+0040) contains a short segment L<<683.0,-73.0>--<670.0,-72.0>>
	* A (U+0041) contains a short segment L<<621.0,0.0>--<624.0,13.0>>
	* A (U+0041) contains a short segment L<<-23.0,13.0>--<-26.0,0.0>>
	* G (U+0047) contains a short segment L<<593.0,287.0>--<592.0,273.0>>
	* K (U+004B) contains a short segment L<<594.0,0.0>--<596.0,13.0>>
	* K (U+004B) contains a short segment L<<690.0,687.0>--<693.0,700.0>> and 68 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_not_italic
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* 🍞 **PASS** No jaggy segments found.
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-Regular.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* less (U+003C): X=578.0,Y=-2.0 (should be at baseline 0?)
	* greater (U+003E): X=122.0,Y=-2.0 (should be at baseline 0?)
	* Q (U+0051): X=477.0,Y=-1.0 (should be at baseline 0?)
	* questiondown (U+00BF): X=491.0,Y=-1.0 (should be at baseline 0?) and questiondown (U+00BF): X=412.0,Y=-1.0 (should be at baseline 0?) [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* ampersand (U+0026) contains a short segment L<<613.0,0.0>--<613.0,12.0>>
	* ampersand (U+0026) contains a short segment L<<610.0,319.0>--<610.0,331.0>>
	* parenleft (U+0028) contains a short segment L<<291.0,-170.0>--<291.0,-158.0>>
	* parenright (U+0029) contains a short segment L<<30.0,690.0>--<30.0,678.0>>
	* seven (U+0037) contains a short segment L<<111.0,12.0>--<111.0,0.0>>
	* at (U+0040) contains a short segment L<<736.0,-66.0>--<724.0,-66.0>>
	* A (U+0041) contains a short segment L<<664.0,0.0>--<664.0,12.0>>
	* A (U+0041) contains a short segment L<<24.0,12.0>--<24.0,0.0>>
	* G (U+0047) contains a short segment L<<597.0,297.0>--<597.0,281.0>>
	* K (U+004B) contains a short segment L<<632.0,0.0>--<632.0,12.0>> and 69 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* 🍞 **PASS** No jaggy segments found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 🍞 **PASS** No semi-horizontal/semi-vertical lines found.
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-SemiBold.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* atilde (U+00E3): X=157.0,Y=699.0 (should be at cap-height 700?)
	* ntilde (U+00F1): X=171.0,Y=699.0 (should be at cap-height 700?)
	* otilde (U+00F5): X=171.0,Y=699.0 (should be at cap-height 700?)
	* tilde (U+02DC): X=41.0,Y=699.0 (should be at cap-height 700?)
	* tildecomb (U+0303): X=41.0,Y=699.0 (should be at cap-height 700?)
	* daggerdbl (U+2021): X=351.0,Y=1.0 (should be at baseline 0?)
	* daggerdbl (U+2021): X=529.0,Y=1.0 (should be at baseline 0?)
	* daggerdbl (U+2021): X=53.0,Y=1.0 (should be at baseline 0?) and daggerdbl (U+2021): X=231.0,Y=1.0 (should be at baseline 0?) [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* ampersand (U+0026) contains a short segment L<<632.0,0.0>--<632.0,15.0>>
	* ampersand (U+0026) contains a short segment L<<629.0,324.0>--<629.0,339.0>>
	* seven (U+0037) contains a short segment L<<94.0,15.0>--<94.0,0.0>>
	* at (U+0040) contains a short segment L<<728.0,-44.0>--<714.0,-44.0>>
	* A (U+0041) contains a short segment L<<671.0,0.0>--<671.0,15.0>>
	* A (U+0041) contains a short segment L<<17.0,15.0>--<17.0,0.0>>
	* G (U+0047) contains a short segment L<<566.0,276.0>--<566.0,264.0>>
	* K (U+004B) contains a short segment L<<648.0,0.0>--<648.0,15.0>>
	* K (U+004B) contains a short segment L<<626.0,685.0>--<626.0,700.0>>
	* V (U+0056) contains a short segment L<<656.0,685.0>--<656.0,700.0>> and 60 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* 🍞 **PASS** No jaggy segments found.
</div></details><details><summary>🍞 <b>PASS:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 🍞 **PASS** No semi-horizontal/semi-vertical lines found.
</div></details><br></div></details><details><summary><b>[74] BRIDigitalText-SemiBoldItalic.ttf</b></summary><div><details><summary>💔 <b>ERROR:</b> List all superfamily filepaths (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/list">com.google.fonts/check/superfamily/list</a>)</summary><div>

>
>This is a merely informative check that lists all sibling families detected by fontbakery.
>
>Only the fontfiles in these directories will be considered in superfamily-level checks.
>
>
* 💔 **ERROR** Failed with IndexError: list index out of range
</div></details><details><summary>⚠ <b>WARN:</b> Check if each glyph has the recommended amount of contours. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/contour_count">com.google.fonts/check/contour_count</a>)</summary><div>

>
>Visually QAing thousands of glyphs by hand is tiring. Most glyphs can only be constructured in a handful of ways. This means a glyph's contour count will only differ slightly amongst different fonts, e.g a 'g' could either be 2 or 3 contours, depending on whether its double story or single story.
>
>However, a quotedbl should have 2 contours, unless the font belongs to a display family.
>
>This check currently does not cover variable fonts because there's plenty of alternative ways of constructing glyphs with multiple outlines for each feature in a VarFont. The expected contour count data for this check is currently optimized for the typical construction of glyphs in static fonts.
>
>
* ⚠ **WARN** This font has a 'Soft Hyphen' character (codepoint 0x00AD) which is supposed to be zero-width and invisible, and is used to mark a hyphenation possibility within a word in the absence of or overriding dictionary hyphenation. It is mostly an obsolete mechanism now, and the character is only included in fonts for legacy codepage coverage. [code: softhyphen]
* ⚠ **WARN** This check inspects the glyph outlines and detects the total number of contours in each of them. The expected values are infered from the typical ammounts of contours observed in a large collection of reference font families. The divergences listed below may simply indicate a significantly different design on some of your glyphs. On the other hand, some of these may flag actual bugs in the font such as glyphs mapped to an incorrect codepoint. Please consider reviewing the design and codepoint assignment of these to make sure they are correct.

The following glyphs do not have the recommended number of contours:

	- Glyph name: uni00AD	Contours detected: 1	Expected: 0 
	- And Glyph name: uni00AD	Contours detected: 1	Expected: 0
 [code: contour-count]
</div></details><details><summary>⚠ <b>WARN:</b> Ensure dotted circle glyph is present and can attach marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/dotted_circle">com.google.fonts/check/dotted_circle</a>)</summary><div>

>
>The dotted circle character (U+25CC) is inserted by shaping engines before mark glyphs which do not have an associated base, especially in the context of broken syllabic clusters.
>
>For fonts containing combining marks, it is recommended that the dotted circle character be included so that these isolated marks can be displayed properly; for fonts supporting complex scripts, this should be considered mandatory.
>
>Additionally, when a dotted circle glyph is present, it should be able to display all marks correctly, meaning that it should contain anchors for all attaching marks.
>
>
* ⚠ **WARN** No dotted circle glyph present [code: missing-dotted-circle]
</div></details><details><summary>⚠ <b>WARN:</b> Are there any misaligned on-curve points? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_alignment_miss">com.google.fonts/check/outline_alignment_miss</a>)</summary><div>

>
>This check heuristically looks for on-curve points which are close to, but do not sit on, significant boundary coordinates. For example, a point which has a Y-coordinate of 1 or -1 might be a misplaced baseline point. As well as the baseline, here we also check for points near the x-height (but only for lower case Latin letters), cap-height, ascender and descender Y coordinates.
>
>Not all such misaligned curve points are a mistake, and sometimes the design may call for points in locations near the boundaries. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported misalignments.
>
>
* ⚠ **WARN** The following glyphs have on-curve points which have potentially incorrect y coordinates:
	* dollar (U+0024): X=222.0,Y=-1.0 (should be at baseline 0?)
	* dollar (U+0024): X=221.0,Y=-1.0 (should be at baseline 0?)
	* percent (U+0025): X=274.0,Y=-1.0 (should be at baseline 0?)
	* percent (U+0025): X=829.0,Y=699.0 (should be at cap-height 700?)
	* g (U+0067): X=398.0,Y=-1.0 (should be at baseline 0?)
	* cent (U+00A2): X=201.0,Y=-2.0 (should be at baseline 0?)
	* atilde (U+00E3): X=226.0,Y=699.0 (should be at cap-height 700?)
	* aring (U+00E5): X=383.0,Y=701.0 (should be at cap-height 700?)
	* ntilde (U+00F1): X=252.0,Y=699.0 (should be at cap-height 700?)
	* otilde (U+00F5): X=248.0,Y=699.0 (should be at cap-height 700?) and 12 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-misalignments]
</div></details><details><summary>⚠ <b>WARN:</b> Are any segments inordinately short? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_short_segments">com.google.fonts/check/outline_short_segments</a>)</summary><div>

>
>This check looks for outline segments which seem particularly short (less than 0.6% of the overall path length).
>
>This check is not run for variable fonts, as they may legitimately have short segments. As this check is liable to generate significant numbers of false positives, it will pass if there are more than 100 reported short segments.
>
>
* ⚠ **WARN** The following glyphs have segments which seem very short:
	* ampersand (U+0026) contains a short segment L<<587.0,0.0>--<588.0,15.0>>
	* ampersand (U+0026) contains a short segment L<<642.0,324.0>--<643.0,339.0>>
	* seven (U+0037) contains a short segment L<<39.0,15.0>--<36.0,0.0>>
	* at (U+0040) contains a short segment L<<681.0,-61.0>--<667.0,-60.0>>
	* A (U+0041) contains a short segment L<<625.0,0.0>--<628.0,15.0>>
	* A (U+0041) contains a short segment L<<-27.0,15.0>--<-30.0,0.0>>
	* G (U+0047) contains a short segment L<<576.0,276.0>--<575.0,264.0>>
	* K (U+004B) contains a short segment L<<601.0,0.0>--<604.0,15.0>>
	* K (U+004B) contains a short segment L<<701.0,685.0>--<703.0,700.0>>
	* V (U+0056) contains a short segment L<<727.0,685.0>--<730.0,700.0>> and 61 more.

Use -F or --full-lists to disable shortening of long lists. [code: found-short-segments]
</div></details><details><summary>⚠ <b>WARN:</b> Do outlines contain any jaggy segments? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_jaggy_segments">com.google.fonts/check/outline_jaggy_segments</a>)</summary><div>

>
>This check heuristically detects outline segments which form a particularly small angle, indicative of an outline error. This may cause false positives in cases such as extreme ink traps, so should be regarded as advisory and backed up by manual inspection.
>
>
* ⚠ **WARN** The following glyphs have jaggy segments:
	* dollar (U+0024): B<<120.0,43.0>-<164.0,13.0>-<222.0,-1.0>>/L<<222.0,-1.0>--<221.0,-1.0>> = 13.570434385161475 [code: found-jaggy-segments]
</div></details><details><summary>💤 <b>SKIP:</b> Check correctness of STAT table strings  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/STAT_strings">com.google.fonts/check/STAT_strings</a>)</summary><div>

>
>On the STAT table, the "Italic" keyword must not be used on AxisValues for variation axes other than 'ital'.
>
>
* 💤 **SKIP** Unfulfilled Conditions: STAT_table
</div></details><details><summary>💤 <b>SKIP:</b> Ensure indic fonts have the Indian Rupee Sign glyph.  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/rupee">com.google.fonts/check/rupee</a>)</summary><div>

>
>Per Bureau of Indian Standards every font supporting one of the official Indian languages needs to include Unicode Character “₹” (U+20B9) Indian Rupee Sign.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_indic_font
</div></details><details><summary>💤 <b>SKIP:</b> Does the font contain chws and vchw features? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/cjk_chws_feature">com.google.fonts/check/cjk_chws_feature</a>)</summary><div>

>
>The W3C recommends the addition of chws and vchw features to CJK fonts to enhance the spacing of glyphs in environments which do not fully support JLREQ layout rules.
>
>The chws_tool utility (https://github.com/googlefonts/chws_tool) can be used to add these features automatically.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cjk_font
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_call_depth">com.adobe.fonts/check/cff_call_depth</a>)</summary><div>

>
>Per "The Type 2 Charstring Format, Technical Note #5177", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> Is the CFF2 subr/gsubr call depth > 10? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff2_call_depth">com.adobe.fonts/check/cff2_call_depth</a>)</summary><div>

>
>Per "The CFF2 CharString Format", the "Subr nesting, stack limit" is 10.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff2
</div></details><details><summary>💤 <b>SKIP:</b> Does the font use deprecated CFF operators or operations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cff.html#com.adobe.fonts/check/cff_deprecated_operators">com.adobe.fonts/check/cff_deprecated_operators</a>)</summary><div>

>
>The 'dotsection' operator and the use of 'endchar' to build accented characters from the Adobe Standard Encoding Character Set ("seac") are deprecated in CFF. Adobe recommends repairing any fonts that use these, especially endchar-as-seac, because a rendering issue was discovered in Microsoft Word with a font that makes use of this operation. The check treats that useage as a FAIL. There are no known ill effects of using dotsection, so that check is a WARN.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> CFF table FontName must match name table ID 6 (PostScript name). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_vs_cff">com.adobe.fonts/check/name/postscript_vs_cff</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should match the FontName string in the 'CFF ' table.
>
>The 'CFF ' table has a lot of information that is duplicated in other tables. This information should be consistent across tables, because there's no guarantee which table an app will get the data from.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_cff
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 400 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wght_coord">com.google.fonts/check/varfont/regular_wght_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>If a variable font has a 'wght' (Weight) axis, then the coordinate of its 'Regular' instance is required to be 400.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be 100 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_wdth_coord">com.google.fonts/check/varfont/regular_wdth_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>If a variable font has a 'wdth' (Width) axis, then the coordinate of its 'Regular' instance is required to be 100.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_wdth_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_slnt_coord">com.google.fonts/check/varfont/regular_slnt_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'slnt' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_slnt
>
>If a variable font has a 'slnt' (Slant) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_slnt_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'ital' (Italic) axis coordinate must be zero on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_ital_coord">com.google.fonts/check/varfont/regular_ital_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'ital' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_ital
>
>If a variable font has a 'ital' (Italic) axis, then the coordinate of its 'Regular' instance is required to be zero.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_ital_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'opsz' (Optical Size) axis coordinate should be between 10 and 16 on the 'Regular' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/regular_opsz_coord">com.google.fonts/check/varfont/regular_opsz_coord</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'opsz' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_opsz
>
>If a variable font has an 'opsz' (Optical Size) axis, then the coordinate of its 'Regular' instance is recommended to be a value in the range 10 to 16.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, regular_opsz_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be 700 on the 'Bold' instance. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/bold_wght_coord">com.google.fonts/check/varfont/bold_wght_coord</a>)</summary><div>

>
>The Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght does not specify a required value for the 'Bold' instance of a variable font.
>
>But Dave Crossland suggested that we should enforce a required value of 700 in this case.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, bold_wght_coord
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wght' (Weight) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wght_valid_range">com.google.fonts/check/varfont/wght_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wght' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wght
>
>On the 'wght' (Weight) axis, the valid coordinate range is 1-1000.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'wdth' (Width) axis coordinate must be within spec range of 1 to 1000 on all instances. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/wdth_valid_range">com.google.fonts/check/varfont/wdth_valid_range</a>)</summary><div>

>
>According to the Open-Type spec's registered design-variation tag 'wdth' available at https://docs.microsoft.com/en-gb/typography/opentype/spec/dvaraxistag_wdth
>
>On the 'wdth' (Width) axis, the valid coordinate range is 1-1000
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> The variable font 'slnt' (Slant) axis coordinate specifies positive values in its range?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/fvar.html#com.google.fonts/check/varfont/slnt_range">com.google.fonts/check/varfont/slnt_range</a>)</summary><div>

>
>The OpenType spec says at https://docs.microsoft.com/en-us/typography/opentype/spec/dvaraxistag_slnt that:
>
>[...] the scale for the Slant axis is interpreted as the angle of slant in counter-clockwise degrees from upright. This means that a typical, right-leaning oblique design will have a negative slant value. This matches the scale used for the italicAngle field in the post table.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font, slnt_axis
</div></details><details><summary>💤 <b>SKIP:</b> All fvar axes have a correspondent Axis Record on STAT table?  (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/stat.html#com.google.fonts/check/varfont/stat_axis_record_for_each_axis">com.google.fonts/check/varfont/stat_axis_record_for_each_axis</a>)</summary><div>

>
>cording to the OpenType spec, there must be an Axis Record for every axis defined in the fvar table.
>
>tps://docs.microsoft.com/en-us/typography/opentype/spec/stat#axis-records
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_variable_font
</div></details><details><summary>💤 <b>SKIP:</b> Do outlines contain any semi-vertical or semi-horizontal lines? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_semi_vertical">com.google.fonts/check/outline_semi_vertical</a>)</summary><div>

>
>This check detects line segments which are nearly, but not quite, exactly horizontal or vertical. Sometimes such lines are created by design, but often they are indicative of a design error.
>
>This check is disabled for italic styles, which often contain nearly-upright lines.
>
>
* 💤 **SKIP** Unfulfilled Conditions: is_not_italic
</div></details><details><summary>💤 <b>SKIP:</b> Check that texts shape as per expectation (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/regression">com.google.fonts/check/shaping/regression</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and compares expected shaping against actual shaping, reporting any differences.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no forbidden glyphs are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/forbidden">com.google.fonts/check/shaping/forbidden</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports if any glyphs are generated in the shaping which should not be produced. (For example, .notdef glyphs, visible viramas, etc.)
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>💤 <b>SKIP:</b> Check that no collisions are found while shaping (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Shaping Checks>.html#com.google.fonts/check/shaping/collides">com.google.fonts/check/shaping/collides</a>)</summary><div>

>
>Fonts with complex layout rules can benefit from regression tests to ensure that the rules are behaving as designed. This checks runs a shaping test suite and reports instances where the glyphs collide in unexpected ways.
>Shaping test suites should be written by the font engineer and referenced in the fontbakery configuration file. For more information about write shaping test files and how to configure fontbakery to read the shaping test suites, see https://simoncozens.github.io/tdd-for-otl/
>
>
* 💤 **SKIP** Shaping test directory not defined in configuration file
</div></details><details><summary>ℹ <b>INFO:</b> Font contains all required tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/required_tables">com.google.fonts/check/required_tables</a>)</summary><div>

>
>Depending on the typeface and coverage of a font, certain tables are recommended for optimum quality. For example, the performance of a non-linear font is improved if the VDMX, LTSH, and hdmx tables are present. Non-monospaced Latin fonts should have a kern table. A gasp table is necessary if a designer wants to influence the sizes at which grayscaling is used under Windows. Etc.
>
>
* ℹ **INFO** This font contains the following optional tables:
	- cvt 
	- fpgm
	- loca
	- prep
	- GPOS
	- GSUB 
	- And gasp [code: optional-tables]
* 🍞 **PASS** Font contains all required tables.
</div></details><details><summary>🍞 <b>PASS:</b> Name table records must not have trailing spaces. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/name/trailing_spaces">com.google.fonts/check/name/trailing_spaces</a>)</summary><div>


* 🍞 **PASS** No trailing spaces on name table entries.
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 usWinAscent & usWinDescent. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/family/win_ascent_and_descent">com.google.fonts/check/family/win_ascent_and_descent</a>)</summary><div>

>
>A font's winAscent and winDescent values should be greater than the head table's yMax, abs(yMin) values. If they are less than these values, clipping can occur on Windows platforms (https://github.com/RedHatBrand/Overpass/issues/33).
>
>If the font includes tall/deep writing systems such as Arabic or Devanagari, the winAscent and winDescent can be greater than the yMax and abs(yMin) to accommodate vowel marks.
>
>When the win Metrics are significantly greater than the upm, the linespacing can appear too loose. To counteract this, enabling the OS/2 fsSelection bit 7 (Use_Typo_Metrics), will force Windows to use the OS/2 typo values instead. This means the font developer can control the linespacing with the typo values, whilst avoiding clipping by setting the win values to values greater than the yMax and abs(yMin).
>
>
* 🍞 **PASS** OS/2 usWinAscent & usWinDescent values look good!
</div></details><details><summary>🍞 <b>PASS:</b> Checking OS/2 Metrics match hhea Metrics. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/os2_metrics_match_hhea">com.google.fonts/check/os2_metrics_match_hhea</a>)</summary><div>

>
>OS/2 and hhea vertical metric values should match. This will produce the same linespacing on Mac, GNU+Linux and Windows.
>
>- Mac OS X uses the hhea values.
>- Windows uses OS/2 or Win, depending on the OS or fsSelection bit value.
>
>When OS/2 and hhea vertical metrics match, the same linespacing results on macOS, GNU+Linux and Windows. Unfortunately as of 2018, Google Fonts has released many fonts with vertical metrics that don't match in this way. When we fix this issue in these existing families, we will create a visible change in line/paragraph layout for either Windows or macOS users, which will upset some of them.
>
>But we have a duty to fix broken stuff, and inconsistent paragraph layout is unacceptably broken when it is possible to avoid it.
>
>If users complain and prefer the old broken version, they have the freedom to take care of their own situation.
>
>
* 🍞 **PASS** OS/2.sTypoAscender/Descender values match hhea.ascent/descent.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with ots-sanitize. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ots">com.google.fonts/check/ots</a>)</summary><div>


* 🍞 **PASS** ots-sanitize passed this file
</div></details><details><summary>🍞 <b>PASS:</b> Font contains '.notdef' as its first glyph? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/mandatory_glyphs">com.google.fonts/check/mandatory_glyphs</a>)</summary><div>

>
>The OpenType specification v1.8.2 recommends that the first glyph is the '.notdef' glyph without a codepoint assigned and with a drawing.
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#glyph-0-the-notdef-glyph
>
>Pre-v1.8, it was recommended that fonts should also contain 'space', 'CR' and '.null' glyphs. This might have been relevant for MacOS 9 applications.
>
>
* 🍞 **PASS** OK
</div></details><details><summary>🍞 <b>PASS:</b> Font contains glyphs for whitespace characters? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphs">com.google.fonts/check/whitespace_glyphs</a>)</summary><div>


* 🍞 **PASS** Font contains glyphs for whitespace characters.
</div></details><details><summary>🍞 <b>PASS:</b> Font has **proper** whitespace glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_glyphnames">com.google.fonts/check/whitespace_glyphnames</a>)</summary><div>

>
>This check enforces adherence to recommended whitespace (codepoints 0020 and 00A0) glyph names according to the Adobe Glyph List.
>
>
* 🍞 **PASS** Font has **AGL recommended** names for whitespace glyphs.
</div></details><details><summary>🍞 <b>PASS:</b> Whitespace glyphs have ink? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/whitespace_ink">com.google.fonts/check/whitespace_ink</a>)</summary><div>


* 🍞 **PASS** There is no whitespace glyph with ink.
</div></details><details><summary>🍞 <b>PASS:</b> Are there unwanted tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unwanted_tables">com.google.fonts/check/unwanted_tables</a>)</summary><div>

>
>Some font editors store source data in their own SFNT tables, and these can sometimes sneak into final release files, which should only have OpenType spec tables.
>
>
* 🍞 **PASS** There are no unwanted tables.
</div></details><details><summary>🍞 <b>PASS:</b> Glyph names are all valid? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/valid_glyphnames">com.google.fonts/check/valid_glyphnames</a>)</summary><div>

>
>Microsoft's recommendations for OpenType Fonts states the following:
>
>'NOTE: The PostScript glyph name must be no longer than 31 characters, include only uppercase or lowercase English letters, European digits, the period or the underscore, i.e. from the set [A-Za-z0-9_.] and should start with a letter, except the special glyph name ".notdef" which starts with a period.'
>
>https://docs.microsoft.com/en-us/typography/opentype/spec/recom#post-table
>
>
>In practice, though, particularly in modern environments, glyph names can be as long as 63 characters.
>According to the "Adobe Glyph List Specification" available at:
>
>https://github.com/adobe-type-tools/agl-specification
>
>
* 🍞 **PASS** Glyph names are all valid.
</div></details><details><summary>🍞 <b>PASS:</b> Font contains unique glyph names? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unique_glyphnames">com.google.fonts/check/unique_glyphnames</a>)</summary><div>

>
>Duplicate glyph names prevent font installation on Mac OS X.
>
>
* 🍞 **PASS** Font contains unique glyph names.
</div></details><details><summary>🍞 <b>PASS:</b> Checking with fontTools.ttx (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/ttx-roundtrip">com.google.fonts/check/ttx-roundtrip</a>)</summary><div>


* 🍞 **PASS** Hey! It all looks good!
</div></details><details><summary>🍞 <b>PASS:</b> Each font in set of sibling families must have the same set of vertical metrics values. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/superfamily/vertical_metrics">com.google.fonts/check/superfamily/vertical_metrics</a>)</summary><div>

>
>We may want all fonts within a super-family (all sibling families) to have the same vertical metrics so their line spacing is consistent across the super-family.
>
>This is an experimental extended version of com.google.fonts/check/family/vertical_metrics and for now it will only result in WARNs.
>
>
* 🍞 **PASS** Vertical metrics are the same across the super-family.
</div></details><details><summary>🍞 <b>PASS:</b> Check font contains no unreachable glyphs (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/unreachable_glyphs">com.google.fonts/check/unreachable_glyphs</a>)</summary><div>

>
>Glyphs are either accessible directly through Unicode codepoints or through substitution rules. Any glyphs not accessible by either of these means are redundant and serve only to increase the font's file size.
>
>
* 🍞 **PASS** Font did not contain any unreachable glyphs
</div></details><details><summary>🍞 <b>PASS:</b> Ensure component transforms do not perform scaling or rotation. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/transformed_components">com.google.fonts/check/transformed_components</a>)</summary><div>

>
>Some families have glyphs which have been constructed by using transformed components e.g the 'u' being constructed from a flipped 'n'.
>
>From a designers point of view, this sounds like a win (less work). However, such approaches can lead to rasterization issues, such as having the 'u' not sitting on the baseline at certain sizes after running the font through ttfautohint.
>
>As of July 2019, Marc Foley observed that ttfautohint assigns cvt values to transformed glyphs as if they are not transformed and the result is they render very badly, and that vttLib does not support flipped components.
>
>When building the font with fontmake, the problem can be fixed by adding this to the command line:
>--filter DecomposeTransformedComponentsFilter
>
>
* 🍞 **PASS** No glyphs had components with scaling or rotation
</div></details><details><summary>🍞 <b>PASS:</b> Ensure no GSUB5/GPOS7 lookups are present. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/universal.html#com.google.fonts/check/gsub5_gpos7">com.google.fonts/check/gsub5_gpos7</a>)</summary><div>

>
>Versions of fonttools >=4.14.0 (19 August 2020) perform an optimisation on chained contextual lookups, expressing GSUB6 as GSUB5 and GPOS8 and GPOS7 where possible (when there are no suffixes/prefixes for all rules in the lookup).
>
>However, makeotf has never generated these lookup types and they are rare in practice. Perhaps before of this, Mac's CoreText shaper does not correctly interpret GPOS7, and certain versions of Windows do not correctly interpret GSUB5, meaning that these lookups will be ignored by the shaper, and fonts containing these lookups will have unintended positioning and substitution errors.
>
>To fix this warning, rebuild the font with a recent version of fonttools.
>
>
* 🍞 **PASS** Font has no GSUB5 or GPOS7 lookups
</div></details><details><summary>🍞 <b>PASS:</b> Check all glyphs have codepoints assigned. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/cmap.html#com.google.fonts/check/all_glyphs_have_codepoints">com.google.fonts/check/all_glyphs_have_codepoints</a>)</summary><div>


* 🍞 **PASS** All glyphs have a codepoint value assigned.
</div></details><details><summary>🍞 <b>PASS:</b> Checking unitsPerEm value is reasonable. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/unitsperem">com.google.fonts/check/unitsperem</a>)</summary><div>

>
>According to the OpenType spec:
>
>The value of unitsPerEm at the head table must be a value between 16 and 16384. Any value in this range is valid.
>
>In fonts that have TrueType outlines, a power of 2 is recommended as this allows performance optimizations in some rasterizers.
>
>But 1000 is a commonly used value. And 2000 may become increasingly more common on Variable Fonts.
>
>
* 🍞 **PASS** The unitsPerEm value (1000) on the 'head' table is reasonable.
</div></details><details><summary>🍞 <b>PASS:</b> Checking font version fields (head and name table). (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/head.html#com.google.fonts/check/font_version">com.google.fonts/check/font_version</a>)</summary><div>


* 🍞 **PASS** All font version fields match.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 xAvgCharWidth is correct. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/xavgcharwidth">com.google.fonts/check/xavgcharwidth</a>)</summary><div>


* 🍞 **PASS** OS/2 xAvgCharWidth value is correct.
</div></details><details><summary>🍞 <b>PASS:</b> Check if OS/2 fsSelection matches head macStyle bold and italic bits. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.adobe.fonts/check/fsselection_matches_macstyle">com.adobe.fonts/check/fsselection_matches_macstyle</a>)</summary><div>

>
>The bold and italic bits in OS/2.fsSelection must match the bold and italic bits in head.macStyle per the OpenType spec.
>
>
* 🍞 **PASS** The OS/2.fsSelection and head.macStyle bold and italic settings match.
</div></details><details><summary>🍞 <b>PASS:</b> Check code page character ranges (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/os2.html#com.google.fonts/check/code_pages">com.google.fonts/check/code_pages</a>)</summary><div>

>
>At least some programs (such as Word and Sublime Text) under Windows 7 do not recognize fonts unless code page bits are properly set on the ulCodePageRange1 (and/or ulCodePageRange2) fields of the OS/2 table.
>
>More specifically, the fonts are selectable in the font menu, but whichever Windows API these applications use considers them unsuitable for any character set, so anything set in these fonts is rendered with a fallback font of Arial.
>
>This check currently does not identify which code pages should be set. Auto-detecting coverage is not trivial since the OpenType specification leaves the interpretation of whether a given code page is "functional" or not open to the font developer to decide.
>
>So here we simply detect as a FAIL when a given font has no code page declared at all.
>
>
* 🍞 **PASS** At least one code page is defined.
</div></details><details><summary>🍞 <b>PASS:</b> Font has correct post table version? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/post.html#com.google.fonts/check/post_table_version">com.google.fonts/check/post_table_version</a>)</summary><div>

>
>Format 2.5 of the 'post' table was deprecated in OpenType 1.3 and should not be used.
>
>According to Thomas Phinney, the possible problem with post format 3 is that under the right combination of circumstances, one can generate PDF from a font with a post format 3 table, and not have accurate backing store for any text that has non-default glyphs for a given codepoint. It will look fine but not be searchable. This can affect Latin text with high-end typography, and some complex script writing systems, especially with
>higher-quality fonts. Those circumstances generally involve creating a PDF by first printing a PostScript stream to disk, and then creating a PDF from that stream without reference to the original source document. There are some workflows where this applies,but these are not common use cases.
>
>Apple recommends against use of post format version 4 as "no longer necessary and should be avoided". Please see the Apple TrueType reference documentation for additional details. 
>https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6post.html
>
>Acceptable post format versions are 2 and 3 for TTF and OTF CFF2 builds, and post format 3 for CFF builds.
>
>
* 🍞 **PASS** Font has an acceptable post format 2.0 table version.
</div></details><details><summary>🍞 <b>PASS:</b> Check name table for empty records. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/empty_records">com.adobe.fonts/check/name/empty_records</a>)</summary><div>

>
>Check the name table for empty records, as this can cause problems in Adobe apps.
>
>
* 🍞 **PASS** No empty name table records found.
</div></details><details><summary>🍞 <b>PASS:</b> Description strings in the name table must not contain copyright info. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/no_copyright_on_description">com.google.fonts/check/name/no_copyright_on_description</a>)</summary><div>


* 🍞 **PASS** Description strings in the name table do not contain any copyright string.
</div></details><details><summary>🍞 <b>PASS:</b> Checking correctness of monospaced metadata. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/monospace">com.google.fonts/check/monospace</a>)</summary><div>

>
>There are various metadata in the OpenType spec to specify if a font is monospaced or not. If the font is not truly monospaced, then no monospaced metadata should be set (as sometimes they mistakenly are...)
>
>Requirements for monospace fonts:
>
>* post.isFixedPitch - "Set to 0 if the font is proportionally spaced, non-zero if the font is not proportionally spaced (monospaced)"
>  www.microsoft.com/typography/otspec/post.htm
>
>* hhea.advanceWidthMax must be correct, meaning no glyph's width value is greater.
>  www.microsoft.com/typography/otspec/hhea.htm
>
>* OS/2.panose.bProportion must be set to 9 (monospace) on latin text fonts.
>
>* OS/2.panose.bSpacing must be set to 3 (monospace) on latin hand written or latin symbol fonts.
>
>* Spec says: "The PANOSE definition contains ten digits each of which currently describes up to sixteen variations. Windows uses bFamilyType, bSerifStyle and bProportion in the font mapper to determine family type. It also uses bProportion to determine if the font is monospaced."
>  www.microsoft.com/typography/otspec/os2.htm#pan
>  monotypecom-test.monotype.de/services/pan2
>
>* OS/2.xAvgCharWidth must be set accurately.
>  "OS/2.xAvgCharWidth is used when rendering monospaced fonts, at least by Windows GDI"
>  http://typedrawers.com/discussion/comment/15397/#Comment_15397
>
>Also we should report an error for glyphs not of average width.
>
>Please also note:
>Thomas Phinney told us that a few years ago (as of December 2019), if you gave a font a monospace flag in Panose, Microsoft Word would ignore the actual advance widths and treat it as monospaced. Source: https://typedrawers.com/discussion/comment/45140/#Comment_45140
>
>
* 🍞 **PASS** Font is not monospaced and all related metadata look good. [code: good]
</div></details><details><summary>🍞 <b>PASS:</b> Does full font name begin with the font family name? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/name/match_familyname_fullfont">com.google.fonts/check/name/match_familyname_fullfont</a>)</summary><div>


* 🍞 **PASS** Full font name begins with the font family name.
</div></details><details><summary>🍞 <b>PASS:</b> Font follows the family naming recommendations? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.google.fonts/check/family_naming_recommendations">com.google.fonts/check/family_naming_recommendations</a>)</summary><div>


* 🍞 **PASS** Font follows the family naming recommendations.
</div></details><details><summary>🍞 <b>PASS:</b> Name table ID 6 (PostScript name) must be consistent across platforms. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/name.html#com.adobe.fonts/check/name/postscript_name_consistency">com.adobe.fonts/check/name/postscript_name_consistency</a>)</summary><div>

>
>The PostScript name entries in the font's 'name' table should be consistent across platforms.
>
>This is the TTF/CFF2 equivalent of the CFF 'name/postscript_vs_cff' check.
>
>
* 🍞 **PASS** Entries in the "name" table for ID 6 (PostScript name) are consistent.
</div></details><details><summary>🍞 <b>PASS:</b> Does the number of glyphs in the loca table match the maxp table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/loca.html#com.google.fonts/check/loca/maxp_num_glyphs">com.google.fonts/check/loca/maxp_num_glyphs</a>)</summary><div>


* 🍞 **PASS** 'loca' table matches numGlyphs in 'maxp' table.
</div></details><details><summary>🍞 <b>PASS:</b> Checking Vertical Metric Linegaps. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/linegaps">com.google.fonts/check/linegaps</a>)</summary><div>


* 🍞 **PASS** OS/2 sTypoLineGap and hhea lineGap are both 0.
</div></details><details><summary>🍞 <b>PASS:</b> MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hhea.html#com.google.fonts/check/maxadvancewidth">com.google.fonts/check/maxadvancewidth</a>)</summary><div>


* 🍞 **PASS** MaxAdvanceWidth is consistent with values in the Hmtx and Hhea tables.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have a DSIG table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/dsig.html#com.google.fonts/check/dsig">com.google.fonts/check/dsig</a>)</summary><div>

>
>Microsoft Office 2013 and below products expect fonts to have a digital signature declared in a DSIG table in order to implement OpenType features. The EOL date for Microsoft Office 2013 products is 4/11/2023. This issue does not impact Microsoft Office 2016 and above products.
>
>As we approach the EOL date, it is now considered better to completely remove the table.
>
>But if you still want your font to support OpenType features on Office 2013, then you may find it handy to add a fake signature on a placeholder DSIG table by running one of the helper scripts provided at https://github.com/googlefonts/gftools
>
>Reference: https://github.com/googlefonts/fontbakery/issues/1845
>
>
* 🍞 **PASS** ok
</div></details><details><summary>🍞 <b>PASS:</b> Space and non-breaking space have the same width? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/hmtx.html#com.google.fonts/check/whitespace_widths">com.google.fonts/check/whitespace_widths</a>)</summary><div>


* 🍞 **PASS** Space and non-breaking space have the same width.
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs in mark glyph class are non-spacing. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_spacing_marks">com.google.fonts/check/gdef_spacing_marks</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class should be non-spacing.
>Spacing glyphs in the GDEF mark glyph class may have incorrect anchor positioning that was only intended for building composite glyphs during design.
>
>
* 🍞 **PASS** Font does not has spacing glyphs in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check mark characters are in GDEF mark glyph class. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_mark_chars">com.google.fonts/check/gdef_mark_chars</a>)</summary><div>

>
>Mark characters should be in the GDEF mark glyph class.
>
>
* 🍞 **PASS** Font does not have mark characters not in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Check GDEF mark glyph class doesn't have characters that are not marks. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gdef.html#com.google.fonts/check/gdef_non_mark_chars">com.google.fonts/check/gdef_non_mark_chars</a>)</summary><div>

>
>Glyphs in the GDEF mark glyph class become non-spacing and may be repositioned if they have mark anchors.
>Only combining mark glyphs should be in that class. Any non-mark glyph must not be in that class, in particular spacing glyphs.
>
>
* 🍞 **PASS** Font does not have non-mark characters in the GDEF mark glyph class.
</div></details><details><summary>🍞 <b>PASS:</b> Does GPOS table have kerning information? This check skips monospaced fonts as defined by post.isFixedPitch value (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/gpos.html#com.google.fonts/check/gpos_kerning_info">com.google.fonts/check/gpos_kerning_info</a>)</summary><div>


* 🍞 **PASS** GPOS table check for kerning information passed.
</div></details><details><summary>🍞 <b>PASS:</b> Is there a usable "kern" table declared in the font? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/kern.html#com.google.fonts/check/kern_table">com.google.fonts/check/kern_table</a>)</summary><div>

>
>Even though all fonts should have their kerning implemented in the GPOS table, there may be kerning info at the kern table as well.
>
>Some applications such as MS PowerPoint require kerning info on the kern table. More specifically, they require a format 0 kern subtable from a kern table version 0 with only glyphs defined in the cmap table, which is the only one that Windows understands (and which is also the simplest and more limited of all the kern subtables).
>
>Google Fonts ingests fonts made for download and use on desktops, and does all web font optimizations in the serving pipeline (using libre libraries that anyone can replicate.)
>
>Ideally, TTFs intended for desktop users (and thus the ones intended for Google Fonts) should have both KERN and GPOS tables.
>
>Given all of the above, we currently treat kerning on a v0 kern table as a good-to-have (but optional) feature.
>
>
* 🍞 **PASS** Font does not declare an optional "kern" table.
</div></details><details><summary>🍞 <b>PASS:</b> Is there any unused data at the end of the glyf table? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_unused_data">com.google.fonts/check/glyf_unused_data</a>)</summary><div>


* 🍞 **PASS** There is no unused data at the end of the glyf table.
</div></details><details><summary>🍞 <b>PASS:</b> Check for points out of bounds. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/points_out_of_bounds">com.google.fonts/check/points_out_of_bounds</a>)</summary><div>


* 🍞 **PASS** All glyph paths have coordinates within bounds!
</div></details><details><summary>🍞 <b>PASS:</b> Check glyphs do not have duplicate components which have the same x,y coordinates. (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/glyf.html#com.google.fonts/check/glyf_non_transformed_duplicate_components">com.google.fonts/check/glyf_non_transformed_duplicate_components</a>)</summary><div>

>
>There have been cases in which fonts had faulty double quote marks, with each of them containing two single quote marks as components with the same x, y coordinates which makes them visually look like single quote marks.
>
>This check ensures that glyphs do not contain duplicate components which have the same x,y coordinates.
>
>
* 🍞 **PASS** Glyphs do not contain duplicate components which have the same x,y coordinates.
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid feature tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_feature_tags">com.google.fonts/check/layout_valid_feature_tags</a>)</summary><div>

>
>Incorrect tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid feature tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid script tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_script_tags">com.google.fonts/check/layout_valid_script_tags</a>)</summary><div>

>
>Incorrect script tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid script tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Does the font have any invalid language tags? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/layout.html#com.google.fonts/check/layout_valid_language_tags">com.google.fonts/check/layout_valid_language_tags</a>)</summary><div>

>
>Incorrect language tags can be indications of typos, leftover debugging code or questionable approaches, or user error in the font editor. Such typos can cause features and language support to fail to work as intended.
>
>
* 🍞 **PASS** No invalid language tags were found
</div></details><details><summary>🍞 <b>PASS:</b> Do any segments have colinear vectors? (<a href="https://font-bakery.readthedocs.io/en/latest/fontbakery/profiles/<Section: Outline Correctness Checks>.html#com.google.fonts/check/outline_colinear_vectors">com.google.fonts/check/outline_colinear_vectors</a>)</summary><div>

>
>This check looks for consecutive line segments which have the same angle. This normally happens if an outline point has been added by accident.
>
>This check is not run for variable fonts, as they may legitimately have colinear vectors.
>
>
* 🍞 **PASS** No colinear vectors found.
</div></details><br></div></details>
### Summary

| 💔 ERROR | 🔥 FAIL | ⚠ WARN | 💤 SKIP | ℹ INFO | 🍞 PASS | 🔎 DEBUG |
|:-----:|:----:|:----:|:----:|:----:|:----:|:----:|
| 12 | 0 | 51 | 247 | 12 | 576 | 0 |
| 1% | 0% | 6% | 28% | 1% | 64% | 0% |
