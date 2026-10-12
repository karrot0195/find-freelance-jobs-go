# Provider Specification: Freelancer Việt

## Target Base URL
- **URL**: `https://freelancerviet.vn/viec-lam-freelance.html`

## Job Item Selectors (Post Item)
Each job card container in the listing page matches the following selector:
- **Container**: `.card-body.p-10`

### Fields Extraction within `.card-body.p-10`:
1. **Title & URL**:
   - **Selector**: `h3 a`
   - **Attributes**: 
     - Text content -> `Job Title`
     - `href` attribute -> `Job URL`

2. **Category**:
   - **Selector**: `ul li:nth-child(1)` (or `ul.li[0]`)
   - **Attributes**: 
     - Text content -> `Job Category`

3. **Price / Budget**:
   - **Selector**: `p.font-size-24`
   - **Attributes**: 
     - Text content -> `Budget / Price`

## Pagination Selector
To navigate through multiple pages of job postings:
- **Next Page Button**: `.pagination .page-item:last a` or `.pagination > page-item:last`
- **Behavior**: Extract the `href` attribute of the next page link to continue scraping subsequent pages until no next link is available.