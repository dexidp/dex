// Package genomics provides access to the Genomics API.
//
// See https://developers.google.com/genomics/v1beta/reference
//
// Usage example:
//
//   import "google.golang.org/api/genomics/v1beta"
//   ...
//   genomicsService, err := genomics.New(oauthHttpClient)
package genomics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/api/googleapi"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Always reference these packages, just in case the auto-generated code
// below doesn't.
var _ = bytes.NewBuffer
var _ = strconv.Itoa
var _ = fmt.Sprintf
var _ = json.NewDecoder
var _ = io.Copy
var _ = url.Parse
var _ = googleapi.Version
var _ = errors.New
var _ = strings.Replace

const apiId = "genomics:v1beta"
const apiName = "genomics"
const apiVersion = "v1beta"
const basePath = "https://www.googleapis.com/genomics/v1beta/"

// OAuth2 scopes used by this API.
const (
	// View and manage your data in Google BigQuery
	BigqueryScope = "https://www.googleapis.com/auth/bigquery"

	// Manage your data in Google Cloud Storage
	DevstorageRead_writeScope = "https://www.googleapis.com/auth/devstorage.read_write"

	// View and manage Genomics data
	GenomicsScope = "https://www.googleapis.com/auth/genomics"

	// View Genomics data
	GenomicsReadonlyScope = "https://www.googleapis.com/auth/genomics.readonly"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Callsets = NewCallsetsService(s)
	s.Datasets = NewDatasetsService(s)
	s.Experimental = NewExperimentalService(s)
	s.Jobs = NewJobsService(s)
	s.Reads = NewReadsService(s)
	s.Readsets = NewReadsetsService(s)
	s.Variants = NewVariantsService(s)
	s.Variantsets = NewVariantsetsService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Callsets *CallsetsService

	Datasets *DatasetsService

	Experimental *ExperimentalService

	Jobs *JobsService

	Reads *ReadsService

	Readsets *ReadsetsService

	Variants *VariantsService

	Variantsets *VariantsetsService
}

func NewCallsetsService(s *Service) *CallsetsService {
	rs := &CallsetsService{s: s}
	return rs
}

type CallsetsService struct {
	s *Service
}

func NewDatasetsService(s *Service) *DatasetsService {
	rs := &DatasetsService{s: s}
	return rs
}

type DatasetsService struct {
	s *Service
}

func NewExperimentalService(s *Service) *ExperimentalService {
	rs := &ExperimentalService{s: s}
	rs.Jobs = NewExperimentalJobsService(s)
	return rs
}

type ExperimentalService struct {
	s *Service

	Jobs *ExperimentalJobsService
}

func NewExperimentalJobsService(s *Service) *ExperimentalJobsService {
	rs := &ExperimentalJobsService{s: s}
	return rs
}

type ExperimentalJobsService struct {
	s *Service
}

func NewJobsService(s *Service) *JobsService {
	rs := &JobsService{s: s}
	return rs
}

type JobsService struct {
	s *Service
}

func NewReadsService(s *Service) *ReadsService {
	rs := &ReadsService{s: s}
	return rs
}

type ReadsService struct {
	s *Service
}

func NewReadsetsService(s *Service) *ReadsetsService {
	rs := &ReadsetsService{s: s}
	rs.Coveragebuckets = NewReadsetsCoveragebucketsService(s)
	return rs
}

type ReadsetsService struct {
	s *Service

	Coveragebuckets *ReadsetsCoveragebucketsService
}

func NewReadsetsCoveragebucketsService(s *Service) *ReadsetsCoveragebucketsService {
	rs := &ReadsetsCoveragebucketsService{s: s}
	return rs
}

type ReadsetsCoveragebucketsService struct {
	s *Service
}

func NewVariantsService(s *Service) *VariantsService {
	rs := &VariantsService{s: s}
	return rs
}

type VariantsService struct {
	s *Service
}

func NewVariantsetsService(s *Service) *VariantsetsService {
	rs := &VariantsetsService{s: s}
	return rs
}

type VariantsetsService struct {
	s *Service
}

type Call struct {
	// CallSetId: The ID of the call set this variant call belongs to.
	CallSetId string `json:"callSetId,omitempty"`

	// CallSetName: The name of the call set this variant call belongs to.
	CallSetName string `json:"callSetName,omitempty"`

	// Genotype: The genotype of this variant call. Each value represents
	// either the value of the referenceBases field or a 1-based index into
	// alternateBases. If a variant had a referenceBases value of T and an
	// alternateBases value of ["A", "C"], and the genotype was [2, 1], that
	// would mean the call represented the heterozygous value CA for this
	// variant. If the genotype was instead [0, 1], the represented value
	// would be TA. Ordering of the genotype values is important if the
	// phaseset is present. If a genotype is not called (that is, a . is
	// present in the GT string) -1 is returned.
	Genotype []int64 `json:"genotype,omitempty"`

	// GenotypeLikelihood: The genotype likelihoods for this variant call.
	// Each array entry represents how likely a specific genotype is for
	// this call. The value ordering is defined by the GL tag in the VCF
	// spec.
	GenotypeLikelihood []float64 `json:"genotypeLikelihood,omitempty"`

	// Info: A map of additional variant call information.
	Info map[string][]string `json:"info,omitempty"`

	// Phaseset: If this field is present, this variant call's genotype
	// ordering implies the phase of the bases and is consistent with any
	// other variant calls in the same reference sequence which have the
	// same phaseset value. When importing data from VCF, if the genotype
	// data was phased but no phase set was specified this field will be set
	// to *.
	Phaseset string `json:"phaseset,omitempty"`
}

type CallSet struct {
	// Created: The date this call set was created in milliseconds from the
	// epoch.
	Created int64 `json:"created,omitempty,string"`

	// Id: The Google generated ID of the call set, immutable.
	Id string `json:"id,omitempty"`

	// Info: A map of additional call set information.
	Info map[string][]string `json:"info,omitempty"`

	// Name: The call set name.
	Name string `json:"name,omitempty"`

	// SampleId: The sample ID this call set corresponds to.
	SampleId string `json:"sampleId,omitempty"`

	// VariantSetIds: The IDs of the variant sets this call set belongs to.
	VariantSetIds []string `json:"variantSetIds,omitempty"`
}

type CoverageBucket struct {
	// MeanCoverage: The average number of reads which are aligned to each
	// individual reference base in this bucket.
	MeanCoverage float64 `json:"meanCoverage,omitempty"`

	// Range: The genomic coordinate range spanned by this bucket.
	Range *GenomicRange `json:"range,omitempty"`
}

type Dataset struct {
	// Id: The Google generated ID of the dataset, immutable.
	Id string `json:"id,omitempty"`

	// IsPublic: Flag indicating whether or not a dataset is publicly
	// viewable. If a dataset is not public, it inherits viewing permissions
	// from its project.
	IsPublic bool `json:"isPublic,omitempty"`

	// Name: The dataset name.
	Name string `json:"name,omitempty"`

	// ProjectId: The Google Developers Console project number that this
	// dataset belongs to.
	ProjectId int64 `json:"projectId,omitempty,string"`
}

type ExperimentalCreateJobRequest struct {
	// Align: Specifies whether or not to run the alignment pipeline. Either
	// align or callVariants must be set.
	Align bool `json:"align,omitempty"`

	// CallVariants: Specifies whether or not to run the variant calling
	// pipeline. Either align or callVariants must be set.
	CallVariants bool `json:"callVariants,omitempty"`

	// GcsOutputPath: Specifies where to copy the results of certain
	// pipelines. This should be in the form of gs://bucket/path.
	GcsOutputPath string `json:"gcsOutputPath,omitempty"`

	// PairedSourceUris: A list of Google Cloud Storage URIs of paired end
	// .fastq files to operate upon. If specified, this represents the
	// second file of each paired .fastq file. The first file of each pair
	// should be specified in sourceUris.
	PairedSourceUris []string `json:"pairedSourceUris,omitempty"`

	// ProjectId: Required. The Google Cloud Project ID with which to
	// associate the request.
	ProjectId int64 `json:"projectId,omitempty,string"`

	// SourceUris: A list of Google Cloud Storage URIs of data files to
	// operate upon. These can be .bam, interleaved .fastq, or paired
	// .fastq. If specifying paired .fastq files, the first of each pair of
	// files should be listed here, and the second of each pair should be
	// listed in pairedSourceUris.
	SourceUris []string `json:"sourceUris,omitempty"`
}

type ExperimentalCreateJobResponse struct {
	// JobId: A job ID that can be used to get status information.
	JobId string `json:"jobId,omitempty"`
}

type ExportReadsetsRequest struct {
	// ExportUri: A Google Cloud Storage URI where the exported BAM file
	// will be created. The currently authenticated user must have write
	// access to the new file location. An error will be returned if the URI
	// already contains data.
	ExportUri string `json:"exportUri,omitempty"`

	// ProjectId: The Google Developers Console project number that owns
	// this export.
	ProjectId int64 `json:"projectId,omitempty,string"`

	// ReadsetIds: The IDs of the readsets to export.
	ReadsetIds []string `json:"readsetIds,omitempty"`

	// ReferenceNames: The reference names to export. If this is not
	// specified, all reference sequences, including unmapped reads, are
	// exported. Use * to export only unmapped reads.
	ReferenceNames []string `json:"referenceNames,omitempty"`
}

type ExportReadsetsResponse struct {
	// JobId: A job ID that can be used to get status information.
	JobId string `json:"jobId,omitempty"`
}

type ExportVariantsRequest struct {
	// BigqueryDataset: The BigQuery dataset to export data to. Note that
	// this is distinct from the Genomics concept of "dataset".
	BigqueryDataset string `json:"bigqueryDataset,omitempty"`

	// BigqueryTable: The BigQuery table to export data to. If the table
	// doesn't exist, it will be created. If it already exists, it will be
	// overwritten.
	BigqueryTable string `json:"bigqueryTable,omitempty"`

	// CallSetIds: If provided, only variant call information from the
	// specified call sets will be exported. By default all variant calls
	// are exported.
	CallSetIds []string `json:"callSetIds,omitempty"`

	// Format: The format for the exported data.
	Format string `json:"format,omitempty"`

	// ProjectId: The Google Cloud project number that owns the destination
	// BigQuery dataset. The caller must have WRITE access to this project.
	// This project will also own the resulting export job.
	ProjectId int64 `json:"projectId,omitempty,string"`

	// VariantSetId: Required. The ID of the variant set that contains
	// variant data which should be exported. The caller must have READ
	// access to this variant set.
	VariantSetId string `json:"variantSetId,omitempty"`
}

type ExportVariantsResponse struct {
	// JobId: A job ID that can be used to get status information.
	JobId string `json:"jobId,omitempty"`
}

type GenomicRange struct {
	// SequenceEnd: The end position of the range on the reference, 1-based
	// exclusive. If specified, sequenceName must also be specified.
	SequenceEnd uint64 `json:"sequenceEnd,omitempty,string"`

	// SequenceName: The reference sequence name, for example chr1, 1, or
	// chrX.
	SequenceName string `json:"sequenceName,omitempty"`

	// SequenceStart: The start position of the range on the reference,
	// 1-based inclusive. If specified, sequenceName must also be specified.
	SequenceStart uint64 `json:"sequenceStart,omitempty,string"`
}

type Header struct {
	// SortingOrder: (SO) Sorting order of alignments.
	SortingOrder string `json:"sortingOrder,omitempty"`

	// Version: (VN) BAM format version.
	Version string `json:"version,omitempty"`
}

type HeaderSection struct {
	// Comments: (@CO) One-line text comments.
	Comments []string `json:"comments,omitempty"`

	// FileUri: [Deprecated] This field is deprecated and will no longer be
	// populated. Please use filename instead.
	FileUri string `json:"fileUri,omitempty"`

	// Filename: The name of the file from which this data was imported.
	Filename string `json:"filename,omitempty"`

	// Headers: (@HD) The header line.
	Headers []*Header `json:"headers,omitempty"`

	// Programs: (@PG) Programs.
	Programs []*Program `json:"programs,omitempty"`

	// ReadGroups: (@RG) Read group.
	ReadGroups []*ReadGroup `json:"readGroups,omitempty"`

	// RefSequences: (@SQ) Reference sequence dictionary.
	RefSequences []*ReferenceSequence `json:"refSequences,omitempty"`
}

type ImportReadsetsRequest struct {
	// DatasetId: Required. The ID of the dataset these readsets will belong
	// to. The caller must have WRITE permissions to this dataset.
	DatasetId string `json:"datasetId,omitempty"`

	// SourceUris: A list of URIs pointing at BAM files in Google Cloud
	// Storage.
	SourceUris []string `json:"sourceUris,omitempty"`
}

type ImportReadsetsResponse struct {
	// JobId: A job ID that can be used to get status information.
	JobId string `json:"jobId,omitempty"`
}

type ImportVariantsRequest struct {
	// Format: The format of the variant data being imported.
	Format string `json:"format,omitempty"`

	// SourceUris: A list of URIs pointing at VCF files in Google Cloud
	// Storage. See the VCF Specification for more details on the input
	// format.
	SourceUris []string `json:"sourceUris,omitempty"`

	// VariantSetId: Required. The variant set to which variant data should
	// be imported.
	VariantSetId string `json:"variantSetId,omitempty"`
}

type ImportVariantsResponse struct {
	// JobId: A job ID that can be used to get status information.
	JobId string `json:"jobId,omitempty"`
}

type Job struct {
	// Created: The date this job was created, in milliseconds from the
	// epoch.
	Created int64 `json:"created,omitempty,string"`

	// Description: A more detailed description of this job's current
	// status.
	Description string `json:"description,omitempty"`

	// Errors: Any errors that occurred during processing.
	Errors []string `json:"errors,omitempty"`

	// Id: The job ID.
	Id string `json:"id,omitempty"`

	// ImportedIds: If this Job represents an import, this field will
	// contain the IDs of the objects that were successfully imported.
	ImportedIds []string `json:"importedIds,omitempty"`

	// ProjectId: The Google Developers Console project number to which this
	// job belongs.
	ProjectId int64 `json:"projectId,omitempty,string"`

	// Request: A summarized representation of the original service request.
	Request *JobRequest `json:"request,omitempty"`

	// Status: The status of this job.
	Status string `json:"status,omitempty"`

	// Warnings: Any warnings that occurred during processing.
	Warnings []string `json:"warnings,omitempty"`
}

type JobRequest struct {
	// Destination: The data destination of the request, for example, a
	// Google BigQuery Table or Dataset ID.
	Destination []string `json:"destination,omitempty"`

	// Source: The data source of the request, for example, a Google Cloud
	// Storage object path or Readset ID.
	Source []string `json:"source,omitempty"`

	// Type: The original request type.
	Type string `json:"type,omitempty"`
}

type ListCoverageBucketsResponse struct {
	// BucketWidth: The length of each coverage bucket in base pairs. Note
	// that buckets at the end of a reference sequence may be shorter. This
	// value is omitted if the bucket width is infinity (the default
	// behaviour, with no range or targetBucketWidth).
	BucketWidth uint64 `json:"bucketWidth,omitempty,string"`

	// CoverageBuckets: The coverage buckets. The list of buckets is sparse;
	// a bucket with 0 overlapping reads is not returned. A bucket never
	// crosses more than one reference sequence. Each bucket has width
	// bucketWidth, unless its end is the end of the reference sequence.
	CoverageBuckets []*CoverageBucket `json:"coverageBuckets,omitempty"`

	// NextPageToken: The continuation token, which is used to page through
	// large result sets. Provide this value in a subsequent request to
	// return the next page of results. This field will be empty if there
	// aren't any additional results.
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type ListDatasetsResponse struct {
	// Datasets: The list of matching Datasets.
	Datasets []*Dataset `json:"datasets,omitempty"`

	// NextPageToken: The continuation token, which is used to page through
	// large result sets. Provide this value in a subsequent request to
	// return the next page of results. This field will be empty if there
	// aren't any additional results.
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type MergeVariantsRequest struct {
	// Variants: The variants to be merged with existing variants.
	Variants []*Variant `json:"variants,omitempty"`
}

type Metadata struct {
	// Description: A textual description of this metadata.
	Description string `json:"description,omitempty"`

	// Id: User-provided ID field, not enforced by this API. Two or more
	// pieces of structured metadata with identical id and key fields are
	// considered equivalent.
	Id string `json:"id,omitempty"`

	// Info: Remaining structured metadata key-value pairs.
	Info map[string][]string `json:"info,omitempty"`

	// Key: The top-level key.
	Key string `json:"key,omitempty"`

	// Number: The number of values that can be included in a field
	// described by this metadata.
	Number string `json:"number,omitempty"`

	// Type: The type of data. Possible types include: Integer, Float, Flag,
	// Character, and String.
	Type string `json:"type,omitempty"`

	// Value: The value field for simple metadata
	Value string `json:"value,omitempty"`
}

type Program struct {
	// CommandLine: (CL) Command line.
	CommandLine string `json:"commandLine,omitempty"`

	// Id: (ID) Program record identifier.
	Id string `json:"id,omitempty"`

	// Name: (PN) Program name.
	Name string `json:"name,omitempty"`

	// PrevProgramId: (PP) Previous program ID.
	PrevProgramId string `json:"prevProgramId,omitempty"`

	// Version: (VN) Program version.
	Version string `json:"version,omitempty"`
}

type Read struct {
	// AlignedBases: The originalBases after the cigar field has been
	// applied. Deletions are represented with '-' and insertions are
	// omitted.
	AlignedBases string `json:"alignedBases,omitempty"`

	// BaseQuality: Represents the quality of each base in this read. Each
	// character represents one base. To get the quality, take the ASCII
	// value of the character and subtract 33. (QUAL)
	BaseQuality string `json:"baseQuality,omitempty"`

	// Cigar: A condensed representation of how this read matches up to the
	// reference. (CIGAR)
	Cigar string `json:"cigar,omitempty"`

	// Flags: Each bit of this number has a different meaning if enabled.
	// See the full BAM spec for more details. (FLAG)
	Flags int64 `json:"flags,omitempty"`

	// Id: The Google generated ID of the read, immutable.
	Id string `json:"id,omitempty"`

	// MappingQuality: A score up to 255 that represents how likely this
	// read's aligned position is to be correct. A higher value is better.
	// (MAPQ)
	MappingQuality int64 `json:"mappingQuality,omitempty"`

	// MatePosition: The 1-based start position of the paired read. (PNEXT)
	MatePosition int64 `json:"matePosition,omitempty"`

	// MateReferenceSequenceName: The name of the sequence that the paired
	// read is aligned to. This is usually the same as
	// referenceSequenceName. (RNEXT)
	MateReferenceSequenceName string `json:"mateReferenceSequenceName,omitempty"`

	// Name: The name of the read. When imported from a BAM file, this is
	// the query template name. (QNAME)
	Name string `json:"name,omitempty"`

	// OriginalBases: The list of bases that this read represents (such as
	// "CATCGA"). (SEQ)
	OriginalBases string `json:"originalBases,omitempty"`

	// Position: The 1-based start position of the aligned read. If the
	// first base starts at the very beginning of the reference sequence,
	// then the position would be '1'. (POS)
	Position int64 `json:"position,omitempty"`

	// ReadsetId: The ID of the readset this read belongs to.
	ReadsetId string `json:"readsetId,omitempty"`

	// ReferenceSequenceName: The name of the sequence that this read is
	// aligned to. This would be, for example, 'X' for the X Chromosome or
	// '20' for Chromosome 20. (RNAME)
	ReferenceSequenceName string `json:"referenceSequenceName,omitempty"`

	// Tags: A map of additional read information. (TAG)
	Tags map[string][]string `json:"tags,omitempty"`

	// TemplateLength: Length of the original piece of DNA that produced
	// both this read and the paired read. (TLEN)
	TemplateLength int64 `json:"templateLength,omitempty"`
}

type ReadGroup struct {
	// Date: (DT) Date the run was produced (ISO8601 date or date/time).
	Date string `json:"date,omitempty"`

	// Description: (DS) Description.
	Description string `json:"description,omitempty"`

	// FlowOrder: (FO) Flow order. The array of nucleotide bases that
	// correspond to the nucleotides used for each flow of each read.
	FlowOrder string `json:"flowOrder,omitempty"`

	// Id: (ID) Read group identifier.
	Id string `json:"id,omitempty"`

	// KeySequence: (KS) The array of nucleotide bases that correspond to
	// the key sequence of each read.
	KeySequence string `json:"keySequence,omitempty"`

	// Library: (LS) Library.
	Library string `json:"library,omitempty"`

	// PlatformUnit: (PU) Platform unit.
	PlatformUnit string `json:"platformUnit,omitempty"`

	// PredictedInsertSize: (PI) Predicted median insert size.
	PredictedInsertSize int64 `json:"predictedInsertSize,omitempty"`

	// ProcessingProgram: (PG) Programs used for processing the read group.
	ProcessingProgram string `json:"processingProgram,omitempty"`

	// Sample: (SM) Sample.
	Sample string `json:"sample,omitempty"`

	// SequencingCenterName: (CN) Name of sequencing center producing the
	// read.
	SequencingCenterName string `json:"sequencingCenterName,omitempty"`

	// SequencingTechnology: (PL) Platform/technology used to produce the
	// reads.
	SequencingTechnology string `json:"sequencingTechnology,omitempty"`
}

type Readset struct {
	// DatasetId: The ID of the dataset this readset belongs to.
	DatasetId string `json:"datasetId,omitempty"`

	// FileData: File information from the original BAM import. See the BAM
	// format specification for additional information on each field.
	FileData []*HeaderSection `json:"fileData,omitempty"`

	// Id: The Google generated ID of the readset, immutable.
	Id string `json:"id,omitempty"`

	// Name: The readset name, typically this is the sample name.
	Name string `json:"name,omitempty"`
}

type ReferenceBound struct {
	// ReferenceName: The reference the bound is associate with.
	ReferenceName string `json:"referenceName,omitempty"`

	// UpperBound: An upper bound (inclusive) on the starting coordinate of
	// any variant in the reference sequence.
	UpperBound int64 `json:"upperBound,omitempty,string"`
}

type ReferenceSequence struct {
	// AssemblyId: (AS) Genome assembly identifier.
	AssemblyId string `json:"assemblyId,omitempty"`

	// Length: (LN) Reference sequence length.
	Length int64 `json:"length,omitempty"`

	// Md5Checksum: (M5) MD5 checksum of the sequence in the uppercase,
	// excluding spaces but including pads as *.
	Md5Checksum string `json:"md5Checksum,omitempty"`

	// Name: (SN) Reference sequence name.
	Name string `json:"name,omitempty"`

	// Species: (SP) Species.
	Species string `json:"species,omitempty"`

	// Uri: (UR) URI of the sequence.
	Uri string `json:"uri,omitempty"`
}

type SearchCallSetsRequest struct {
	// Name: Only return call sets for which a substring of the name matches
	// this string.
	Name string `json:"name,omitempty"`

	// PageSize: The maximum number of call sets to return.
	PageSize int64 `json:"pageSize,omitempty"`

	// PageToken: The continuation token, which is used to page through
	// large result sets. To get the next page of results, set this
	// parameter to the value of nextPageToken from the previous response.
	PageToken string `json:"pageToken,omitempty"`

	// VariantSetIds: Restrict the query to call sets within the given
	// variant sets. At least one ID must be provided.
	VariantSetIds []string `json:"variantSetIds,omitempty"`
}

type SearchCallSetsResponse struct {
	// CallSets: The list of matching call sets.
	CallSets []*CallSet `json:"callSets,omitempty"`

	// NextPageToken: The continuation token, which is used to page through
	// large result sets. Provide this value in a subsequent request to
	// return the next page of results. This field will be empty if there
	// aren't any additional results.
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type SearchJobsRequest struct {
	// CreatedAfter: If specified, only jobs created on or after this date,
	// given in milliseconds since Unix epoch, will be returned.
	CreatedAfter int64 `json:"createdAfter,omitempty,string"`

	// CreatedBefore: If specified, only jobs created prior to this date,
	// given in milliseconds since Unix epoch, will be returned.
	CreatedBefore int64 `json:"createdBefore,omitempty,string"`

	// MaxResults: Specifies the number of results to return in a single
	// page. Defaults to 128. The maximum value is 256.
	MaxResults uint64 `json:"maxResults,omitempty,string"`

	// PageToken: The continuation token which is used to page through large
	// result sets. To get the next page of results, set this parameter to
	// the value of the nextPageToken from the previous response.
	PageToken string `json:"pageToken,omitempty"`

	// ProjectId: Required. Only return jobs which belong to this Google
	// Developers
	ProjectId int64 `json:"projectId,omitempty,string"`

	// Status: Only return jobs which have a matching status.
	Status []string `json:"status,omitempty"`
}

type SearchJobsResponse struct {
	// Jobs: The list of jobs results, ordered newest to oldest.
	Jobs []*Job `json:"jobs,omitempty"`

	// NextPageToken: The continuation token which is used to page through
	// large result sets. Provide this value is a subsequent request to
	// return the next page of results. This field will be empty if there
	// are no more results.
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type SearchReadsRequest struct {
	// MaxResults: Specifies number of results to return in a single page.
	// If unspecified, it will default to 256. The maximum value is 2048.
	MaxResults uint64 `json:"maxResults,omitempty,string"`

	// PageToken: The continuation token, which is used to page through
	// large result sets. To get the next page of results, set this
	// parameter to the value of nextPageToken from the previous response.
	PageToken string `json:"pageToken,omitempty"`

	// ReadsetIds: The readsets within which to search for reads. At least
	// one readset ID must be provided. All specified readsets must be
	// aligned against a common set of reference sequences; this defines the
	// genomic coordinates for the query.
	ReadsetIds []string `json:"readsetIds,omitempty"`

	// SequenceEnd: The end position (1-based, inclusive) of the target
	// range. If specified, sequenceName must also be specified. Defaults to
	// the end of the target reference sequence, if any.
	SequenceEnd uint64 `json:"sequenceEnd,omitempty,string"`

	// SequenceName: Restricts the results to a particular reference
	// sequence such as 1, chr1, or X. The set of valid references sequences
	// depends on the readsets specified. If set to *, only unmapped Reads
	// are returned.
	SequenceName string `json:"sequenceName,omitempty"`

	// SequenceStart: The start position (1-based, inclusive) of the target
	// range. If specified, sequenceName must also be specified. Defaults to
	// the start of the target reference sequence, if any.
	SequenceStart uint64 `json:"sequenceStart,omitempty,string"`
}

type SearchReadsResponse struct {
	// NextPageToken: The continuation token, which is used to page through
	// large result sets. Provide this value in a subsequent request to
	// return the next page of results. This field will be empty if there
	// aren't any additional results.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// Reads: The list of matching Reads. The resulting Reads are sorted by
	// position; the smallest positions are returned first. Unmapped reads,
	// which have no position, are returned last and are further sorted
	// alphabetically by name.
	Reads []*Read `json:"reads,omitempty"`
}

type SearchReadsetsRequest struct {
	// DatasetIds: Restricts this query to readsets within the given
	// datasets. At least one ID must be provided.
	DatasetIds []string `json:"datasetIds,omitempty"`

	// MaxResults: Specifies number of results to return in a single page.
	// If unspecified, it will default to 128. The maximum value is 1024.
	MaxResults uint64 `json:"maxResults,omitempty,string"`

	// Name: Only return readsets for which a substring of the name matches
	// this string.
	Name string `json:"name,omitempty"`

	// PageToken: The continuation token, which is used to page through
	// large result sets. To get the next page of results, set this
	// parameter to the value of nextPageToken from the previous response.
	PageToken string `json:"pageToken,omitempty"`
}

type SearchReadsetsResponse struct {
	// NextPageToken: The continuation token, which is used to page through
	// large result sets. Provide this value in a subsequent request to
	// return the next page of results. This field will be empty if there
	// aren't any additional results.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// Readsets: The list of matching Readsets.
	Readsets []*Readset `json:"readsets,omitempty"`
}

type SearchVariantSetsRequest struct {
	// DatasetIds: Exactly one dataset ID must be provided here. Only
	// variant sets which belong to this dataset will be returned.
	DatasetIds []string `json:"datasetIds,omitempty"`

	// PageSize: The maximum number of variant sets to return in a request.
	PageSize int64 `json:"pageSize,omitempty"`

	// PageToken: The continuation token, which is used to page through
	// large result sets. To get the next page of results, set this
	// parameter to the value of nextPageToken from the previous response.
	PageToken string `json:"pageToken,omitempty"`
}

type SearchVariantSetsResponse struct {
	// NextPageToken: The continuation token, which is used to page through
	// large result sets. Provide this value in a subsequent request to
	// return the next page of results. This field will be empty if there
	// aren't any additional results.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// VariantSets: The variant sets belonging to the requested dataset.
	VariantSets []*VariantSet `json:"variantSets,omitempty"`
}

type SearchVariantsRequest struct {
	// CallSetIds: Only return variant calls which belong to call sets with
	// these ids. Leaving this blank returns all variant calls. If a variant
	// has no calls belonging to any of these call sets, it won't be
	// returned at all. Currently, variants with no calls from any call set
	// will never be returned.
	CallSetIds []string `json:"callSetIds,omitempty"`

	// End: Required. The end of the window (0-based, exclusive) for which
	// overlapping variants should be returned.
	End int64 `json:"end,omitempty,string"`

	// MaxCalls: The maximum number of calls to return. However, at least
	// one variant will always be returned, even if it has more calls than
	// this limit.
	MaxCalls int64 `json:"maxCalls,omitempty"`

	// PageSize: The maximum number of variants to return.
	PageSize int64 `json:"pageSize,omitempty"`

	// PageToken: The continuation token, which is used to page through
	// large result sets. To get the next page of results, set this
	// parameter to the value of nextPageToken from the previous response.
	PageToken string `json:"pageToken,omitempty"`

	// ReferenceName: Required. Only return variants in this reference
	// sequence.
	ReferenceName string `json:"referenceName,omitempty"`

	// Start: Required. The beginning of the window (0-based, inclusive) for
	// which overlapping variants should be returned.
	Start int64 `json:"start,omitempty,string"`

	// VariantName: Only return variants which have exactly this name.
	VariantName string `json:"variantName,omitempty"`

	// VariantSetIds: Exactly one variant set ID must be provided. Only
	// variants from this variant set will be returned.
	VariantSetIds []string `json:"variantSetIds,omitempty"`
}

type SearchVariantsResponse struct {
	// NextPageToken: The continuation token, which is used to page through
	// large result sets. Provide this value in a subsequent request to
	// return the next page of results. This field will be empty if there
	// aren't any additional results.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// Variants: The list of matching Variants.
	Variants []*Variant `json:"variants,omitempty"`
}

type Variant struct {
	// AlternateBases: The bases that appear instead of the reference bases.
	AlternateBases []string `json:"alternateBases,omitempty"`

	// Calls: The variant calls for this particular variant. Each one
	// represents the determination of genotype with respect to this
	// variant.
	Calls []*Call `json:"calls,omitempty"`

	// Created: The date this variant was created, in milliseconds from the
	// epoch.
	Created int64 `json:"created,omitempty,string"`

	// End: The end position (0-based) of this variant. This corresponds to
	// the first base after the last base in the reference allele. So, the
	// length of the reference allele is (end - start). This is useful for
	// variants that don't explicitly give alternate bases, for example
	// large deletions.
	End int64 `json:"end,omitempty,string"`

	// Filter: A list of filters (normally quality filters) this variant has
	// failed. PASS indicates this variant has passed all filters.
	Filter []string `json:"filter,omitempty"`

	// Id: The Google generated ID of the variant, immutable.
	Id string `json:"id,omitempty"`

	// Info: A map of additional variant information.
	Info map[string][]string `json:"info,omitempty"`

	// Names: Names for the variant, for example a RefSNP ID.
	Names []string `json:"names,omitempty"`

	// Quality: A measure of how likely this variant is to be real. A higher
	// value is better.
	Quality float64 `json:"quality,omitempty"`

	// ReferenceBases: The reference bases for this variant. They start at
	// the given position.
	ReferenceBases string `json:"referenceBases,omitempty"`

	// ReferenceName: The reference on which this variant occurs. (such as
	// chr20 or X)
	ReferenceName string `json:"referenceName,omitempty"`

	// Start: The position at which this variant occurs (0-based). This
	// corresponds to the first base of the string of reference bases.
	Start int64 `json:"start,omitempty,string"`

	// VariantSetId: The ID of the variant set this variant belongs to.
	VariantSetId string `json:"variantSetId,omitempty"`
}

type VariantSet struct {
	// DatasetId: The dataset to which this variant set belongs. Immutable.
	DatasetId string `json:"datasetId,omitempty"`

	// Id: The Google-generated ID of the variant set. Immutable.
	Id string `json:"id,omitempty"`

	// Metadata: The metadata associated with this variant set.
	Metadata []*Metadata `json:"metadata,omitempty"`

	// ReferenceBounds: A list of all references used by the variants in a
	// variant set with associated coordinate upper bounds for each one.
	ReferenceBounds []*ReferenceBound `json:"referenceBounds,omitempty"`
}

// method id "genomics.callsets.create":

type CallsetsCreateCall struct {
	s       *Service
	callset *CallSet
	opt_    map[string]interface{}
}

// Create: Creates a new call set.
func (r *CallsetsService) Create(callset *CallSet) *CallsetsCreateCall {
	c := &CallsetsCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.callset = callset
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *CallsetsCreateCall) Fields(s ...googleapi.Field) *CallsetsCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *CallsetsCreateCall) Do() (*CallSet, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.callset)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "callsets")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *CallSet
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a new call set.",
	//   "httpMethod": "POST",
	//   "id": "genomics.callsets.create",
	//   "path": "callsets",
	//   "request": {
	//     "$ref": "CallSet"
	//   },
	//   "response": {
	//     "$ref": "CallSet"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.callsets.delete":

type CallsetsDeleteCall struct {
	s         *Service
	callSetId string
	opt_      map[string]interface{}
}

// Delete: Deletes a call set.
func (r *CallsetsService) Delete(callSetId string) *CallsetsDeleteCall {
	c := &CallsetsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.callSetId = callSetId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *CallsetsDeleteCall) Fields(s ...googleapi.Field) *CallsetsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *CallsetsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "callsets/{callSetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"callSetId": c.callSetId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Deletes a call set.",
	//   "httpMethod": "DELETE",
	//   "id": "genomics.callsets.delete",
	//   "parameterOrder": [
	//     "callSetId"
	//   ],
	//   "parameters": {
	//     "callSetId": {
	//       "description": "The ID of the call set to be deleted.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "callsets/{callSetId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.callsets.get":

type CallsetsGetCall struct {
	s         *Service
	callSetId string
	opt_      map[string]interface{}
}

// Get: Gets a call set by ID.
func (r *CallsetsService) Get(callSetId string) *CallsetsGetCall {
	c := &CallsetsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.callSetId = callSetId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *CallsetsGetCall) Fields(s ...googleapi.Field) *CallsetsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *CallsetsGetCall) Do() (*CallSet, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "callsets/{callSetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"callSetId": c.callSetId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *CallSet
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a call set by ID.",
	//   "httpMethod": "GET",
	//   "id": "genomics.callsets.get",
	//   "parameterOrder": [
	//     "callSetId"
	//   ],
	//   "parameters": {
	//     "callSetId": {
	//       "description": "The ID of the call set.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "callsets/{callSetId}",
	//   "response": {
	//     "$ref": "CallSet"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.callsets.patch":

type CallsetsPatchCall struct {
	s         *Service
	callSetId string
	callset   *CallSet
	opt_      map[string]interface{}
}

// Patch: Updates a call set. This method supports patch semantics.
func (r *CallsetsService) Patch(callSetId string, callset *CallSet) *CallsetsPatchCall {
	c := &CallsetsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.callSetId = callSetId
	c.callset = callset
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *CallsetsPatchCall) Fields(s ...googleapi.Field) *CallsetsPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *CallsetsPatchCall) Do() (*CallSet, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.callset)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "callsets/{callSetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"callSetId": c.callSetId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *CallSet
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a call set. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "genomics.callsets.patch",
	//   "parameterOrder": [
	//     "callSetId"
	//   ],
	//   "parameters": {
	//     "callSetId": {
	//       "description": "The ID of the call set to be updated.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "callsets/{callSetId}",
	//   "request": {
	//     "$ref": "CallSet"
	//   },
	//   "response": {
	//     "$ref": "CallSet"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.callsets.search":

type CallsetsSearchCall struct {
	s                     *Service
	searchcallsetsrequest *SearchCallSetsRequest
	opt_                  map[string]interface{}
}

// Search: Gets a list of call sets matching the criteria.
//
// Implements
// GlobalAllianceApi.searchCallSets.
func (r *CallsetsService) Search(searchcallsetsrequest *SearchCallSetsRequest) *CallsetsSearchCall {
	c := &CallsetsSearchCall{s: r.s, opt_: make(map[string]interface{})}
	c.searchcallsetsrequest = searchcallsetsrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *CallsetsSearchCall) Fields(s ...googleapi.Field) *CallsetsSearchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *CallsetsSearchCall) Do() (*SearchCallSetsResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.searchcallsetsrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "callsets/search")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *SearchCallSetsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a list of call sets matching the criteria.\n\nImplements GlobalAllianceApi.searchCallSets.",
	//   "httpMethod": "POST",
	//   "id": "genomics.callsets.search",
	//   "path": "callsets/search",
	//   "request": {
	//     "$ref": "SearchCallSetsRequest"
	//   },
	//   "response": {
	//     "$ref": "SearchCallSetsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.callsets.update":

type CallsetsUpdateCall struct {
	s         *Service
	callSetId string
	callset   *CallSet
	opt_      map[string]interface{}
}

// Update: Updates a call set.
func (r *CallsetsService) Update(callSetId string, callset *CallSet) *CallsetsUpdateCall {
	c := &CallsetsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.callSetId = callSetId
	c.callset = callset
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *CallsetsUpdateCall) Fields(s ...googleapi.Field) *CallsetsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *CallsetsUpdateCall) Do() (*CallSet, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.callset)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "callsets/{callSetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"callSetId": c.callSetId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *CallSet
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a call set.",
	//   "httpMethod": "PUT",
	//   "id": "genomics.callsets.update",
	//   "parameterOrder": [
	//     "callSetId"
	//   ],
	//   "parameters": {
	//     "callSetId": {
	//       "description": "The ID of the call set to be updated.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "callsets/{callSetId}",
	//   "request": {
	//     "$ref": "CallSet"
	//   },
	//   "response": {
	//     "$ref": "CallSet"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.datasets.create":

type DatasetsCreateCall struct {
	s       *Service
	dataset *Dataset
	opt_    map[string]interface{}
}

// Create: Creates a new dataset.
func (r *DatasetsService) Create(dataset *Dataset) *DatasetsCreateCall {
	c := &DatasetsCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.dataset = dataset
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatasetsCreateCall) Fields(s ...googleapi.Field) *DatasetsCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatasetsCreateCall) Do() (*Dataset, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.dataset)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "datasets")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Dataset
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a new dataset.",
	//   "httpMethod": "POST",
	//   "id": "genomics.datasets.create",
	//   "path": "datasets",
	//   "request": {
	//     "$ref": "Dataset"
	//   },
	//   "response": {
	//     "$ref": "Dataset"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.datasets.delete":

type DatasetsDeleteCall struct {
	s         *Service
	datasetId string
	opt_      map[string]interface{}
}

// Delete: Deletes a dataset.
func (r *DatasetsService) Delete(datasetId string) *DatasetsDeleteCall {
	c := &DatasetsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.datasetId = datasetId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatasetsDeleteCall) Fields(s ...googleapi.Field) *DatasetsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatasetsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "datasets/{datasetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"datasetId": c.datasetId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Deletes a dataset.",
	//   "httpMethod": "DELETE",
	//   "id": "genomics.datasets.delete",
	//   "parameterOrder": [
	//     "datasetId"
	//   ],
	//   "parameters": {
	//     "datasetId": {
	//       "description": "The ID of the dataset to be deleted.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "datasets/{datasetId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.datasets.get":

type DatasetsGetCall struct {
	s         *Service
	datasetId string
	opt_      map[string]interface{}
}

// Get: Gets a dataset by ID.
func (r *DatasetsService) Get(datasetId string) *DatasetsGetCall {
	c := &DatasetsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.datasetId = datasetId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatasetsGetCall) Fields(s ...googleapi.Field) *DatasetsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatasetsGetCall) Do() (*Dataset, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "datasets/{datasetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"datasetId": c.datasetId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Dataset
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a dataset by ID.",
	//   "httpMethod": "GET",
	//   "id": "genomics.datasets.get",
	//   "parameterOrder": [
	//     "datasetId"
	//   ],
	//   "parameters": {
	//     "datasetId": {
	//       "description": "The ID of the dataset.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "datasets/{datasetId}",
	//   "response": {
	//     "$ref": "Dataset"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.datasets.list":

type DatasetsListCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// List: Lists all datasets.
func (r *DatasetsService) List() *DatasetsListCall {
	c := &DatasetsListCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// MaxResults sets the optional parameter "maxResults": The maximum
// number of results returned by this request.
func (c *DatasetsListCall) MaxResults(maxResults uint64) *DatasetsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": The continuation
// token, which is used to page through large result sets. To get the
// next page of results, set this parameter to the value of
// nextPageToken from the previous response.
func (c *DatasetsListCall) PageToken(pageToken string) *DatasetsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// ProjectId sets the optional parameter "projectId": Only return
// datasets which belong to this Google Developers Console project. Only
// accepts project numbers. Returns all public projects if no project
// number is specified.
func (c *DatasetsListCall) ProjectId(projectId int64) *DatasetsListCall {
	c.opt_["projectId"] = projectId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatasetsListCall) Fields(s ...googleapi.Field) *DatasetsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatasetsListCall) Do() (*ListDatasetsResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["projectId"]; ok {
		params.Set("projectId", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "datasets")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ListDatasetsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all datasets.",
	//   "httpMethod": "GET",
	//   "id": "genomics.datasets.list",
	//   "parameters": {
	//     "maxResults": {
	//       "default": "50",
	//       "description": "The maximum number of results returned by this request.",
	//       "format": "uint64",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "pageToken": {
	//       "description": "The continuation token, which is used to page through large result sets. To get the next page of results, set this parameter to the value of nextPageToken from the previous response.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "projectId": {
	//       "description": "Only return datasets which belong to this Google Developers Console project. Only accepts project numbers. Returns all public projects if no project number is specified.",
	//       "format": "int64",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "datasets",
	//   "response": {
	//     "$ref": "ListDatasetsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.datasets.patch":

type DatasetsPatchCall struct {
	s         *Service
	datasetId string
	dataset   *Dataset
	opt_      map[string]interface{}
}

// Patch: Updates a dataset. This method supports patch semantics.
func (r *DatasetsService) Patch(datasetId string, dataset *Dataset) *DatasetsPatchCall {
	c := &DatasetsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.datasetId = datasetId
	c.dataset = dataset
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatasetsPatchCall) Fields(s ...googleapi.Field) *DatasetsPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatasetsPatchCall) Do() (*Dataset, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.dataset)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "datasets/{datasetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"datasetId": c.datasetId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Dataset
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a dataset. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "genomics.datasets.patch",
	//   "parameterOrder": [
	//     "datasetId"
	//   ],
	//   "parameters": {
	//     "datasetId": {
	//       "description": "The ID of the dataset to be updated.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "datasets/{datasetId}",
	//   "request": {
	//     "$ref": "Dataset"
	//   },
	//   "response": {
	//     "$ref": "Dataset"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.datasets.undelete":

type DatasetsUndeleteCall struct {
	s         *Service
	datasetId string
	opt_      map[string]interface{}
}

// Undelete: Undeletes a dataset by restoring a dataset which was
// deleted via this API. This operation is only possible for a week
// after the deletion occurred.
func (r *DatasetsService) Undelete(datasetId string) *DatasetsUndeleteCall {
	c := &DatasetsUndeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.datasetId = datasetId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatasetsUndeleteCall) Fields(s ...googleapi.Field) *DatasetsUndeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatasetsUndeleteCall) Do() (*Dataset, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "datasets/{datasetId}/undelete")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"datasetId": c.datasetId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Dataset
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Undeletes a dataset by restoring a dataset which was deleted via this API. This operation is only possible for a week after the deletion occurred.",
	//   "httpMethod": "POST",
	//   "id": "genomics.datasets.undelete",
	//   "parameterOrder": [
	//     "datasetId"
	//   ],
	//   "parameters": {
	//     "datasetId": {
	//       "description": "The ID of the dataset to be undeleted.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "datasets/{datasetId}/undelete",
	//   "response": {
	//     "$ref": "Dataset"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.datasets.update":

type DatasetsUpdateCall struct {
	s         *Service
	datasetId string
	dataset   *Dataset
	opt_      map[string]interface{}
}

// Update: Updates a dataset.
func (r *DatasetsService) Update(datasetId string, dataset *Dataset) *DatasetsUpdateCall {
	c := &DatasetsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.datasetId = datasetId
	c.dataset = dataset
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatasetsUpdateCall) Fields(s ...googleapi.Field) *DatasetsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatasetsUpdateCall) Do() (*Dataset, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.dataset)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "datasets/{datasetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"datasetId": c.datasetId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Dataset
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a dataset.",
	//   "httpMethod": "PUT",
	//   "id": "genomics.datasets.update",
	//   "parameterOrder": [
	//     "datasetId"
	//   ],
	//   "parameters": {
	//     "datasetId": {
	//       "description": "The ID of the dataset to be updated.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "datasets/{datasetId}",
	//   "request": {
	//     "$ref": "Dataset"
	//   },
	//   "response": {
	//     "$ref": "Dataset"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.experimental.jobs.create":

type ExperimentalJobsCreateCall struct {
	s                            *Service
	experimentalcreatejobrequest *ExperimentalCreateJobRequest
	opt_                         map[string]interface{}
}

// Create: Creates and asynchronously runs an ad-hoc job. This is an
// experimental call and may be removed or changed at any time.
func (r *ExperimentalJobsService) Create(experimentalcreatejobrequest *ExperimentalCreateJobRequest) *ExperimentalJobsCreateCall {
	c := &ExperimentalJobsCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.experimentalcreatejobrequest = experimentalcreatejobrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ExperimentalJobsCreateCall) Fields(s ...googleapi.Field) *ExperimentalJobsCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ExperimentalJobsCreateCall) Do() (*ExperimentalCreateJobResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.experimentalcreatejobrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "experimental/jobs/create")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ExperimentalCreateJobResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates and asynchronously runs an ad-hoc job. This is an experimental call and may be removed or changed at any time.",
	//   "httpMethod": "POST",
	//   "id": "genomics.experimental.jobs.create",
	//   "path": "experimental/jobs/create",
	//   "request": {
	//     "$ref": "ExperimentalCreateJobRequest"
	//   },
	//   "response": {
	//     "$ref": "ExperimentalCreateJobResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/devstorage.read_write",
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.jobs.cancel":

type JobsCancelCall struct {
	s     *Service
	jobId string
	opt_  map[string]interface{}
}

// Cancel: Cancels a job by ID. Note that it is possible for partial
// results to be generated and stored for cancelled jobs.
func (r *JobsService) Cancel(jobId string) *JobsCancelCall {
	c := &JobsCancelCall{s: r.s, opt_: make(map[string]interface{})}
	c.jobId = jobId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *JobsCancelCall) Fields(s ...googleapi.Field) *JobsCancelCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *JobsCancelCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "jobs/{jobId}/cancel")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"jobId": c.jobId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Cancels a job by ID. Note that it is possible for partial results to be generated and stored for cancelled jobs.",
	//   "httpMethod": "POST",
	//   "id": "genomics.jobs.cancel",
	//   "parameterOrder": [
	//     "jobId"
	//   ],
	//   "parameters": {
	//     "jobId": {
	//       "description": "Required. The ID of the job.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "jobs/{jobId}/cancel",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.jobs.get":

type JobsGetCall struct {
	s     *Service
	jobId string
	opt_  map[string]interface{}
}

// Get: Gets a job by ID.
func (r *JobsService) Get(jobId string) *JobsGetCall {
	c := &JobsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.jobId = jobId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *JobsGetCall) Fields(s ...googleapi.Field) *JobsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *JobsGetCall) Do() (*Job, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "jobs/{jobId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"jobId": c.jobId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Job
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a job by ID.",
	//   "httpMethod": "GET",
	//   "id": "genomics.jobs.get",
	//   "parameterOrder": [
	//     "jobId"
	//   ],
	//   "parameters": {
	//     "jobId": {
	//       "description": "Required. The ID of the job.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "jobs/{jobId}",
	//   "response": {
	//     "$ref": "Job"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.jobs.search":

type JobsSearchCall struct {
	s                 *Service
	searchjobsrequest *SearchJobsRequest
	opt_              map[string]interface{}
}

// Search: Gets a list of jobs matching the criteria.
func (r *JobsService) Search(searchjobsrequest *SearchJobsRequest) *JobsSearchCall {
	c := &JobsSearchCall{s: r.s, opt_: make(map[string]interface{})}
	c.searchjobsrequest = searchjobsrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *JobsSearchCall) Fields(s ...googleapi.Field) *JobsSearchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *JobsSearchCall) Do() (*SearchJobsResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.searchjobsrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "jobs/search")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *SearchJobsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a list of jobs matching the criteria.",
	//   "httpMethod": "POST",
	//   "id": "genomics.jobs.search",
	//   "path": "jobs/search",
	//   "request": {
	//     "$ref": "SearchJobsRequest"
	//   },
	//   "response": {
	//     "$ref": "SearchJobsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.reads.search":

type ReadsSearchCall struct {
	s                  *Service
	searchreadsrequest *SearchReadsRequest
	opt_               map[string]interface{}
}

// Search: Gets a list of reads for one or more readsets. Reads search
// operates over a genomic coordinate space of reference sequence &
// position defined over the reference sequences to which the requested
// readsets are aligned. If a target positional range is specified,
// search returns all reads whose alignment to the reference genome
// overlap the range. A query which specifies only readset IDs yields
// all reads in those readsets, including unmapped reads. All reads
// returned (including reads on subsequent pages) are ordered by genomic
// coordinate (reference sequence & position). Reads with equivalent
// genomic coordinates are returned in a deterministic order.
func (r *ReadsService) Search(searchreadsrequest *SearchReadsRequest) *ReadsSearchCall {
	c := &ReadsSearchCall{s: r.s, opt_: make(map[string]interface{})}
	c.searchreadsrequest = searchreadsrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ReadsSearchCall) Fields(s ...googleapi.Field) *ReadsSearchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ReadsSearchCall) Do() (*SearchReadsResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.searchreadsrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "reads/search")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *SearchReadsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a list of reads for one or more readsets. Reads search operates over a genomic coordinate space of reference sequence \u0026 position defined over the reference sequences to which the requested readsets are aligned. If a target positional range is specified, search returns all reads whose alignment to the reference genome overlap the range. A query which specifies only readset IDs yields all reads in those readsets, including unmapped reads. All reads returned (including reads on subsequent pages) are ordered by genomic coordinate (reference sequence \u0026 position). Reads with equivalent genomic coordinates are returned in a deterministic order.",
	//   "httpMethod": "POST",
	//   "id": "genomics.reads.search",
	//   "path": "reads/search",
	//   "request": {
	//     "$ref": "SearchReadsRequest"
	//   },
	//   "response": {
	//     "$ref": "SearchReadsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.readsets.delete":

type ReadsetsDeleteCall struct {
	s         *Service
	readsetId string
	opt_      map[string]interface{}
}

// Delete: Deletes a readset.
func (r *ReadsetsService) Delete(readsetId string) *ReadsetsDeleteCall {
	c := &ReadsetsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.readsetId = readsetId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ReadsetsDeleteCall) Fields(s ...googleapi.Field) *ReadsetsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ReadsetsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "readsets/{readsetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"readsetId": c.readsetId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Deletes a readset.",
	//   "httpMethod": "DELETE",
	//   "id": "genomics.readsets.delete",
	//   "parameterOrder": [
	//     "readsetId"
	//   ],
	//   "parameters": {
	//     "readsetId": {
	//       "description": "The ID of the readset to be deleted. The caller must have WRITE permissions to the dataset associated with this readset.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "readsets/{readsetId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.readsets.export":

type ReadsetsExportCall struct {
	s                     *Service
	exportreadsetsrequest *ExportReadsetsRequest
	opt_                  map[string]interface{}
}

// Export: Exports readsets to a BAM file in Google Cloud Storage. Note
// that currently there may be some differences between exported BAM
// files and the original BAM file at the time of import. In particular,
// comments in the input file header will not be preserved, and some
// custom tags will be converted to strings.
func (r *ReadsetsService) Export(exportreadsetsrequest *ExportReadsetsRequest) *ReadsetsExportCall {
	c := &ReadsetsExportCall{s: r.s, opt_: make(map[string]interface{})}
	c.exportreadsetsrequest = exportreadsetsrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ReadsetsExportCall) Fields(s ...googleapi.Field) *ReadsetsExportCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ReadsetsExportCall) Do() (*ExportReadsetsResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.exportreadsetsrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "readsets/export")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ExportReadsetsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Exports readsets to a BAM file in Google Cloud Storage. Note that currently there may be some differences between exported BAM files and the original BAM file at the time of import. In particular, comments in the input file header will not be preserved, and some custom tags will be converted to strings.",
	//   "httpMethod": "POST",
	//   "id": "genomics.readsets.export",
	//   "path": "readsets/export",
	//   "request": {
	//     "$ref": "ExportReadsetsRequest"
	//   },
	//   "response": {
	//     "$ref": "ExportReadsetsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/devstorage.read_write",
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.readsets.get":

type ReadsetsGetCall struct {
	s         *Service
	readsetId string
	opt_      map[string]interface{}
}

// Get: Gets a readset by ID.
func (r *ReadsetsService) Get(readsetId string) *ReadsetsGetCall {
	c := &ReadsetsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.readsetId = readsetId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ReadsetsGetCall) Fields(s ...googleapi.Field) *ReadsetsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ReadsetsGetCall) Do() (*Readset, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "readsets/{readsetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"readsetId": c.readsetId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Readset
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a readset by ID.",
	//   "httpMethod": "GET",
	//   "id": "genomics.readsets.get",
	//   "parameterOrder": [
	//     "readsetId"
	//   ],
	//   "parameters": {
	//     "readsetId": {
	//       "description": "The ID of the readset.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "readsets/{readsetId}",
	//   "response": {
	//     "$ref": "Readset"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.readsets.import":

type ReadsetsImportCall struct {
	s                     *Service
	importreadsetsrequest *ImportReadsetsRequest
	opt_                  map[string]interface{}
}

// Import: Creates readsets by asynchronously importing the provided
// information. Note that currently comments in the input file header
// are not imported and some custom tags will be converted to strings,
// rather than preserving tag types. The caller must have WRITE
// permissions to the dataset.
func (r *ReadsetsService) Import(importreadsetsrequest *ImportReadsetsRequest) *ReadsetsImportCall {
	c := &ReadsetsImportCall{s: r.s, opt_: make(map[string]interface{})}
	c.importreadsetsrequest = importreadsetsrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ReadsetsImportCall) Fields(s ...googleapi.Field) *ReadsetsImportCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ReadsetsImportCall) Do() (*ImportReadsetsResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.importreadsetsrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "readsets/import")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ImportReadsetsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates readsets by asynchronously importing the provided information. Note that currently comments in the input file header are not imported and some custom tags will be converted to strings, rather than preserving tag types. The caller must have WRITE permissions to the dataset.",
	//   "httpMethod": "POST",
	//   "id": "genomics.readsets.import",
	//   "path": "readsets/import",
	//   "request": {
	//     "$ref": "ImportReadsetsRequest"
	//   },
	//   "response": {
	//     "$ref": "ImportReadsetsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/devstorage.read_write",
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.readsets.patch":

type ReadsetsPatchCall struct {
	s         *Service
	readsetId string
	readset   *Readset
	opt_      map[string]interface{}
}

// Patch: Updates a readset. This method supports patch semantics.
func (r *ReadsetsService) Patch(readsetId string, readset *Readset) *ReadsetsPatchCall {
	c := &ReadsetsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.readsetId = readsetId
	c.readset = readset
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ReadsetsPatchCall) Fields(s ...googleapi.Field) *ReadsetsPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ReadsetsPatchCall) Do() (*Readset, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.readset)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "readsets/{readsetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"readsetId": c.readsetId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Readset
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a readset. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "genomics.readsets.patch",
	//   "parameterOrder": [
	//     "readsetId"
	//   ],
	//   "parameters": {
	//     "readsetId": {
	//       "description": "The ID of the readset to be updated. The caller must have WRITE permissions to the dataset associated with this readset.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "readsets/{readsetId}",
	//   "request": {
	//     "$ref": "Readset"
	//   },
	//   "response": {
	//     "$ref": "Readset"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.readsets.search":

type ReadsetsSearchCall struct {
	s                     *Service
	searchreadsetsrequest *SearchReadsetsRequest
	opt_                  map[string]interface{}
}

// Search: Gets a list of readsets matching the criteria.
func (r *ReadsetsService) Search(searchreadsetsrequest *SearchReadsetsRequest) *ReadsetsSearchCall {
	c := &ReadsetsSearchCall{s: r.s, opt_: make(map[string]interface{})}
	c.searchreadsetsrequest = searchreadsetsrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ReadsetsSearchCall) Fields(s ...googleapi.Field) *ReadsetsSearchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ReadsetsSearchCall) Do() (*SearchReadsetsResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.searchreadsetsrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "readsets/search")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *SearchReadsetsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a list of readsets matching the criteria.",
	//   "httpMethod": "POST",
	//   "id": "genomics.readsets.search",
	//   "path": "readsets/search",
	//   "request": {
	//     "$ref": "SearchReadsetsRequest"
	//   },
	//   "response": {
	//     "$ref": "SearchReadsetsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.readsets.update":

type ReadsetsUpdateCall struct {
	s         *Service
	readsetId string
	readset   *Readset
	opt_      map[string]interface{}
}

// Update: Updates a readset.
func (r *ReadsetsService) Update(readsetId string, readset *Readset) *ReadsetsUpdateCall {
	c := &ReadsetsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.readsetId = readsetId
	c.readset = readset
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ReadsetsUpdateCall) Fields(s ...googleapi.Field) *ReadsetsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ReadsetsUpdateCall) Do() (*Readset, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.readset)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "readsets/{readsetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"readsetId": c.readsetId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Readset
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a readset.",
	//   "httpMethod": "PUT",
	//   "id": "genomics.readsets.update",
	//   "parameterOrder": [
	//     "readsetId"
	//   ],
	//   "parameters": {
	//     "readsetId": {
	//       "description": "The ID of the readset to be updated. The caller must have WRITE permissions to the dataset associated with this readset.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "readsets/{readsetId}",
	//   "request": {
	//     "$ref": "Readset"
	//   },
	//   "response": {
	//     "$ref": "Readset"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.readsets.coveragebuckets.list":

type ReadsetsCoveragebucketsListCall struct {
	s         *Service
	readsetId string
	opt_      map[string]interface{}
}

// List: Lists fixed width coverage buckets for a readset, each of which
// correspond to a range of a reference sequence. Each bucket summarizes
// coverage information across its corresponding genomic range. Coverage
// is defined as the number of reads which are aligned to a given base
// in the reference sequence. Coverage buckets are available at various
// bucket widths, enabling various coverage "zoom levels". The caller
// must have READ permissions for the target readset.
func (r *ReadsetsCoveragebucketsService) List(readsetId string) *ReadsetsCoveragebucketsListCall {
	c := &ReadsetsCoveragebucketsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.readsetId = readsetId
	return c
}

// MaxResults sets the optional parameter "maxResults": The maximum
// number of results to return in a single page. If unspecified,
// defaults to 1024. The maximum value is 2048.
func (c *ReadsetsCoveragebucketsListCall) MaxResults(maxResults uint64) *ReadsetsCoveragebucketsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": The continuation
// token, which is used to page through large result sets. To get the
// next page of results, set this parameter to the value of
// nextPageToken from the previous response.
func (c *ReadsetsCoveragebucketsListCall) PageToken(pageToken string) *ReadsetsCoveragebucketsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// RangeSequenceEnd sets the optional parameter "range.sequenceEnd": The
// end position of the range on the reference, 1-based exclusive. If
// specified, sequenceName must also be specified.
func (c *ReadsetsCoveragebucketsListCall) RangeSequenceEnd(rangeSequenceEnd uint64) *ReadsetsCoveragebucketsListCall {
	c.opt_["range.sequenceEnd"] = rangeSequenceEnd
	return c
}

// RangeSequenceName sets the optional parameter "range.sequenceName":
// The reference sequence name, for example chr1, 1, or chrX.
func (c *ReadsetsCoveragebucketsListCall) RangeSequenceName(rangeSequenceName string) *ReadsetsCoveragebucketsListCall {
	c.opt_["range.sequenceName"] = rangeSequenceName
	return c
}

// RangeSequenceStart sets the optional parameter "range.sequenceStart":
// The start position of the range on the reference, 1-based inclusive.
// If specified, sequenceName must also be specified.
func (c *ReadsetsCoveragebucketsListCall) RangeSequenceStart(rangeSequenceStart uint64) *ReadsetsCoveragebucketsListCall {
	c.opt_["range.sequenceStart"] = rangeSequenceStart
	return c
}

// TargetBucketWidth sets the optional parameter "targetBucketWidth":
// The desired width of each reported coverage bucket in base pairs.
// This will be rounded down to the nearest precomputed bucket width;
// the value of which is returned as bucketWidth in the response.
// Defaults to infinity (each bucket spans an entire reference sequence)
// or the length of the target range, if specified. The smallest
// precomputed bucketWidth is currently 2048 base pairs; this is subject
// to change.
func (c *ReadsetsCoveragebucketsListCall) TargetBucketWidth(targetBucketWidth uint64) *ReadsetsCoveragebucketsListCall {
	c.opt_["targetBucketWidth"] = targetBucketWidth
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ReadsetsCoveragebucketsListCall) Fields(s ...googleapi.Field) *ReadsetsCoveragebucketsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ReadsetsCoveragebucketsListCall) Do() (*ListCoverageBucketsResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["range.sequenceEnd"]; ok {
		params.Set("range.sequenceEnd", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["range.sequenceName"]; ok {
		params.Set("range.sequenceName", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["range.sequenceStart"]; ok {
		params.Set("range.sequenceStart", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["targetBucketWidth"]; ok {
		params.Set("targetBucketWidth", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "readsets/{readsetId}/coveragebuckets")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"readsetId": c.readsetId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ListCoverageBucketsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists fixed width coverage buckets for a readset, each of which correspond to a range of a reference sequence. Each bucket summarizes coverage information across its corresponding genomic range. Coverage is defined as the number of reads which are aligned to a given base in the reference sequence. Coverage buckets are available at various bucket widths, enabling various coverage \"zoom levels\". The caller must have READ permissions for the target readset.",
	//   "httpMethod": "GET",
	//   "id": "genomics.readsets.coveragebuckets.list",
	//   "parameterOrder": [
	//     "readsetId"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "default": "1024",
	//       "description": "The maximum number of results to return in a single page. If unspecified, defaults to 1024. The maximum value is 2048.",
	//       "format": "uint64",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "pageToken": {
	//       "description": "The continuation token, which is used to page through large result sets. To get the next page of results, set this parameter to the value of nextPageToken from the previous response.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "range.sequenceEnd": {
	//       "description": "The end position of the range on the reference, 1-based exclusive. If specified, sequenceName must also be specified.",
	//       "format": "uint64",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "range.sequenceName": {
	//       "description": "The reference sequence name, for example chr1, 1, or chrX.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "range.sequenceStart": {
	//       "description": "The start position of the range on the reference, 1-based inclusive. If specified, sequenceName must also be specified.",
	//       "format": "uint64",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "readsetId": {
	//       "description": "Required. The ID of the readset over which coverage is requested.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "targetBucketWidth": {
	//       "description": "The desired width of each reported coverage bucket in base pairs. This will be rounded down to the nearest precomputed bucket width; the value of which is returned as bucketWidth in the response. Defaults to infinity (each bucket spans an entire reference sequence) or the length of the target range, if specified. The smallest precomputed bucketWidth is currently 2048 base pairs; this is subject to change.",
	//       "format": "uint64",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "readsets/{readsetId}/coveragebuckets",
	//   "response": {
	//     "$ref": "ListCoverageBucketsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.variants.create":

type VariantsCreateCall struct {
	s       *Service
	variant *Variant
	opt_    map[string]interface{}
}

// Create: Creates a new variant.
func (r *VariantsService) Create(variant *Variant) *VariantsCreateCall {
	c := &VariantsCreateCall{s: r.s, opt_: make(map[string]interface{})}
	c.variant = variant
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsCreateCall) Fields(s ...googleapi.Field) *VariantsCreateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsCreateCall) Do() (*Variant, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.variant)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variants")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Variant
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a new variant.",
	//   "httpMethod": "POST",
	//   "id": "genomics.variants.create",
	//   "path": "variants",
	//   "request": {
	//     "$ref": "Variant"
	//   },
	//   "response": {
	//     "$ref": "Variant"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.variants.delete":

type VariantsDeleteCall struct {
	s         *Service
	variantId string
	opt_      map[string]interface{}
}

// Delete: Deletes a variant.
func (r *VariantsService) Delete(variantId string) *VariantsDeleteCall {
	c := &VariantsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.variantId = variantId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsDeleteCall) Fields(s ...googleapi.Field) *VariantsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variants/{variantId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"variantId": c.variantId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Deletes a variant.",
	//   "httpMethod": "DELETE",
	//   "id": "genomics.variants.delete",
	//   "parameterOrder": [
	//     "variantId"
	//   ],
	//   "parameters": {
	//     "variantId": {
	//       "description": "The ID of the variant to be deleted.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "variants/{variantId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.variants.export":

type VariantsExportCall struct {
	s                     *Service
	exportvariantsrequest *ExportVariantsRequest
	opt_                  map[string]interface{}
}

// Export: Exports variant data to an external destination.
func (r *VariantsService) Export(exportvariantsrequest *ExportVariantsRequest) *VariantsExportCall {
	c := &VariantsExportCall{s: r.s, opt_: make(map[string]interface{})}
	c.exportvariantsrequest = exportvariantsrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsExportCall) Fields(s ...googleapi.Field) *VariantsExportCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsExportCall) Do() (*ExportVariantsResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.exportvariantsrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variants/export")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ExportVariantsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Exports variant data to an external destination.",
	//   "httpMethod": "POST",
	//   "id": "genomics.variants.export",
	//   "path": "variants/export",
	//   "request": {
	//     "$ref": "ExportVariantsRequest"
	//   },
	//   "response": {
	//     "$ref": "ExportVariantsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/bigquery",
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.variants.get":

type VariantsGetCall struct {
	s         *Service
	variantId string
	opt_      map[string]interface{}
}

// Get: Gets a variant by ID.
func (r *VariantsService) Get(variantId string) *VariantsGetCall {
	c := &VariantsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.variantId = variantId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsGetCall) Fields(s ...googleapi.Field) *VariantsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsGetCall) Do() (*Variant, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variants/{variantId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"variantId": c.variantId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Variant
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a variant by ID.",
	//   "httpMethod": "GET",
	//   "id": "genomics.variants.get",
	//   "parameterOrder": [
	//     "variantId"
	//   ],
	//   "parameters": {
	//     "variantId": {
	//       "description": "The ID of the variant.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "variants/{variantId}",
	//   "response": {
	//     "$ref": "Variant"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.variants.import":

type VariantsImportCall struct {
	s                     *Service
	importvariantsrequest *ImportVariantsRequest
	opt_                  map[string]interface{}
}

// Import: Creates variant data by asynchronously importing the provided
// information. The variants for import will be merged with any existing
// data and each other according to the behavior of mergeVariants. In
// particular, this means for merged VCF variants that have conflicting
// INFO fields, some data will be arbitrarily discarded. As a special
// case, for single-sample VCF files, QUAL and FILTER fields will be
// moved to the call level; these are sometimes interpreted in a
// call-specific context. Imported VCF headers are appended to the
// metadata already in a variant set.
func (r *VariantsService) Import(importvariantsrequest *ImportVariantsRequest) *VariantsImportCall {
	c := &VariantsImportCall{s: r.s, opt_: make(map[string]interface{})}
	c.importvariantsrequest = importvariantsrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsImportCall) Fields(s ...googleapi.Field) *VariantsImportCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsImportCall) Do() (*ImportVariantsResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.importvariantsrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variants/import")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ImportVariantsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates variant data by asynchronously importing the provided information. The variants for import will be merged with any existing data and each other according to the behavior of mergeVariants. In particular, this means for merged VCF variants that have conflicting INFO fields, some data will be arbitrarily discarded. As a special case, for single-sample VCF files, QUAL and FILTER fields will be moved to the call level; these are sometimes interpreted in a call-specific context. Imported VCF headers are appended to the metadata already in a variant set.",
	//   "httpMethod": "POST",
	//   "id": "genomics.variants.import",
	//   "path": "variants/import",
	//   "request": {
	//     "$ref": "ImportVariantsRequest"
	//   },
	//   "response": {
	//     "$ref": "ImportVariantsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/devstorage.read_write",
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.variants.search":

type VariantsSearchCall struct {
	s                     *Service
	searchvariantsrequest *SearchVariantsRequest
	opt_                  map[string]interface{}
}

// Search: Gets a list of variants matching the criteria.
//
// Implements
// GlobalAllianceApi.searchVariants.
func (r *VariantsService) Search(searchvariantsrequest *SearchVariantsRequest) *VariantsSearchCall {
	c := &VariantsSearchCall{s: r.s, opt_: make(map[string]interface{})}
	c.searchvariantsrequest = searchvariantsrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsSearchCall) Fields(s ...googleapi.Field) *VariantsSearchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsSearchCall) Do() (*SearchVariantsResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.searchvariantsrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variants/search")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *SearchVariantsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a list of variants matching the criteria.\n\nImplements GlobalAllianceApi.searchVariants.",
	//   "httpMethod": "POST",
	//   "id": "genomics.variants.search",
	//   "path": "variants/search",
	//   "request": {
	//     "$ref": "SearchVariantsRequest"
	//   },
	//   "response": {
	//     "$ref": "SearchVariantsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.variants.update":

type VariantsUpdateCall struct {
	s         *Service
	variantId string
	variant   *Variant
	opt_      map[string]interface{}
}

// Update: Updates a variant's names and info fields. All other
// modifications are silently ignored. Returns the modified variant
// without its calls.
func (r *VariantsService) Update(variantId string, variant *Variant) *VariantsUpdateCall {
	c := &VariantsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.variantId = variantId
	c.variant = variant
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsUpdateCall) Fields(s ...googleapi.Field) *VariantsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsUpdateCall) Do() (*Variant, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.variant)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variants/{variantId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"variantId": c.variantId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Variant
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a variant's names and info fields. All other modifications are silently ignored. Returns the modified variant without its calls.",
	//   "httpMethod": "PUT",
	//   "id": "genomics.variants.update",
	//   "parameterOrder": [
	//     "variantId"
	//   ],
	//   "parameters": {
	//     "variantId": {
	//       "description": "The ID of the variant to be updated.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "variants/{variantId}",
	//   "request": {
	//     "$ref": "Variant"
	//   },
	//   "response": {
	//     "$ref": "Variant"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.variantsets.delete":

type VariantsetsDeleteCall struct {
	s            *Service
	variantSetId string
	opt_         map[string]interface{}
}

// Delete: Deletes the contents of a variant set. The variant set object
// is not deleted.
func (r *VariantsetsService) Delete(variantSetId string) *VariantsetsDeleteCall {
	c := &VariantsetsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.variantSetId = variantSetId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsetsDeleteCall) Fields(s ...googleapi.Field) *VariantsetsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsetsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variantsets/{variantSetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"variantSetId": c.variantSetId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Deletes the contents of a variant set. The variant set object is not deleted.",
	//   "httpMethod": "DELETE",
	//   "id": "genomics.variantsets.delete",
	//   "parameterOrder": [
	//     "variantSetId"
	//   ],
	//   "parameters": {
	//     "variantSetId": {
	//       "description": "The ID of the variant set to be deleted.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "variantsets/{variantSetId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.variantsets.get":

type VariantsetsGetCall struct {
	s            *Service
	variantSetId string
	opt_         map[string]interface{}
}

// Get: Gets a variant set by ID.
func (r *VariantsetsService) Get(variantSetId string) *VariantsetsGetCall {
	c := &VariantsetsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.variantSetId = variantSetId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsetsGetCall) Fields(s ...googleapi.Field) *VariantsetsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsetsGetCall) Do() (*VariantSet, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variantsets/{variantSetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"variantSetId": c.variantSetId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *VariantSet
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a variant set by ID.",
	//   "httpMethod": "GET",
	//   "id": "genomics.variantsets.get",
	//   "parameterOrder": [
	//     "variantSetId"
	//   ],
	//   "parameters": {
	//     "variantSetId": {
	//       "description": "Required. The ID of the variant set.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "variantsets/{variantSetId}",
	//   "response": {
	//     "$ref": "VariantSet"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.variantsets.mergeVariants":

type VariantsetsMergeVariantsCall struct {
	s                    *Service
	variantSetId         string
	mergevariantsrequest *MergeVariantsRequest
	opt_                 map[string]interface{}
}

// MergeVariants: Merges the given variants with existing variants. Each
// variant will be merged with an existing variant that matches its
// reference sequence, start, end, reference bases, and alternative
// bases. If no such variant exists, a new one will be created.
//
// When
// variants are merged, the call information from the new variant is
// added to the existing variant, and other fields (such as key/value
// pairs) are discarded.
func (r *VariantsetsService) MergeVariants(variantSetId string, mergevariantsrequest *MergeVariantsRequest) *VariantsetsMergeVariantsCall {
	c := &VariantsetsMergeVariantsCall{s: r.s, opt_: make(map[string]interface{})}
	c.variantSetId = variantSetId
	c.mergevariantsrequest = mergevariantsrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsetsMergeVariantsCall) Fields(s ...googleapi.Field) *VariantsetsMergeVariantsCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsetsMergeVariantsCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.mergevariantsrequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variantsets/{variantSetId}/mergeVariants")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"variantSetId": c.variantSetId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Merges the given variants with existing variants. Each variant will be merged with an existing variant that matches its reference sequence, start, end, reference bases, and alternative bases. If no such variant exists, a new one will be created.\n\nWhen variants are merged, the call information from the new variant is added to the existing variant, and other fields (such as key/value pairs) are discarded.",
	//   "httpMethod": "POST",
	//   "id": "genomics.variantsets.mergeVariants",
	//   "parameterOrder": [
	//     "variantSetId"
	//   ],
	//   "parameters": {
	//     "variantSetId": {
	//       "description": "The destination variant set.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "variantsets/{variantSetId}/mergeVariants",
	//   "request": {
	//     "$ref": "MergeVariantsRequest"
	//   }
	// }

}

// method id "genomics.variantsets.patch":

type VariantsetsPatchCall struct {
	s            *Service
	variantSetId string
	variantset   *VariantSet
	opt_         map[string]interface{}
}

// Patch: Updates a variant set's metadata. All other modifications are
// silently ignored. This method supports patch semantics.
func (r *VariantsetsService) Patch(variantSetId string, variantset *VariantSet) *VariantsetsPatchCall {
	c := &VariantsetsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.variantSetId = variantSetId
	c.variantset = variantset
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsetsPatchCall) Fields(s ...googleapi.Field) *VariantsetsPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsetsPatchCall) Do() (*VariantSet, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.variantset)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variantsets/{variantSetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"variantSetId": c.variantSetId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *VariantSet
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a variant set's metadata. All other modifications are silently ignored. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "genomics.variantsets.patch",
	//   "parameterOrder": [
	//     "variantSetId"
	//   ],
	//   "parameters": {
	//     "variantSetId": {
	//       "description": "The ID of the variant to be updated.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "variantsets/{variantSetId}",
	//   "request": {
	//     "$ref": "VariantSet"
	//   },
	//   "response": {
	//     "$ref": "VariantSet"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}

// method id "genomics.variantsets.search":

type VariantsetsSearchCall struct {
	s                        *Service
	searchvariantsetsrequest *SearchVariantSetsRequest
	opt_                     map[string]interface{}
}

// Search: Returns a list of all variant sets matching search
// criteria.
//
// Implements GlobalAllianceApi.searchVariantSets.
func (r *VariantsetsService) Search(searchvariantsetsrequest *SearchVariantSetsRequest) *VariantsetsSearchCall {
	c := &VariantsetsSearchCall{s: r.s, opt_: make(map[string]interface{})}
	c.searchvariantsetsrequest = searchvariantsetsrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsetsSearchCall) Fields(s ...googleapi.Field) *VariantsetsSearchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsetsSearchCall) Do() (*SearchVariantSetsResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.searchvariantsetsrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variantsets/search")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *SearchVariantSetsResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Returns a list of all variant sets matching search criteria.\n\nImplements GlobalAllianceApi.searchVariantSets.",
	//   "httpMethod": "POST",
	//   "id": "genomics.variantsets.search",
	//   "path": "variantsets/search",
	//   "request": {
	//     "$ref": "SearchVariantSetsRequest"
	//   },
	//   "response": {
	//     "$ref": "SearchVariantSetsResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics",
	//     "https://www.googleapis.com/auth/genomics.readonly"
	//   ]
	// }

}

// method id "genomics.variantsets.update":

type VariantsetsUpdateCall struct {
	s            *Service
	variantSetId string
	variantset   *VariantSet
	opt_         map[string]interface{}
}

// Update: Updates a variant set's metadata. All other modifications are
// silently ignored.
func (r *VariantsetsService) Update(variantSetId string, variantset *VariantSet) *VariantsetsUpdateCall {
	c := &VariantsetsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.variantSetId = variantSetId
	c.variantset = variantset
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *VariantsetsUpdateCall) Fields(s ...googleapi.Field) *VariantsetsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *VariantsetsUpdateCall) Do() (*VariantSet, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.variantset)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "variantsets/{variantSetId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"variantSetId": c.variantSetId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *VariantSet
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a variant set's metadata. All other modifications are silently ignored.",
	//   "httpMethod": "PUT",
	//   "id": "genomics.variantsets.update",
	//   "parameterOrder": [
	//     "variantSetId"
	//   ],
	//   "parameters": {
	//     "variantSetId": {
	//       "description": "The ID of the variant to be updated.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "variantsets/{variantSetId}",
	//   "request": {
	//     "$ref": "VariantSet"
	//   },
	//   "response": {
	//     "$ref": "VariantSet"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/genomics"
	//   ]
	// }

}
