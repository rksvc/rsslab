import {
  AnchorButton,
  Button,
  ButtonVariant,
  Classes,
  Code,
  Dialog,
  DialogBody,
  DialogFooter,
  Divider,
  FormGroup,
  HTMLSelect,
  InputGroup,
  Intent,
  Section,
  SectionCard,
  TextArea,
} from '@blueprintjs/core'
import {
  type CSSProperties,
  type Dispatch,
  type HTMLAttributes,
  type SetStateAction,
  useRef,
  useState,
} from 'react'
import { ExternalLink } from 'react-feather'
import { useMyContext } from './Context.tsx'
import type { Feed, Transformer } from './types.ts'
import { compareTitle, param, parseFeedLink, xfetch } from './utils.ts'

type Param = {
  value: string
  setValue: Dispatch<SetStateAction<string>>
  key: string
  desc?: string | JSX.Element
  placeholder?: string
  multiline?: boolean
}

export function NewFeedDialog({
  isOpen,
  setIsOpen,
}: {
  isOpen: boolean
  setIsOpen: Dispatch<SetStateAction<boolean>>
}) {
  const { setFeeds, updateStatus, setSelected, foldersWithFeeds, selected, feedsById } = useMyContext()
  const [loading, setLoading] = useState(false)
  const [feedLink, setFeedLink] = useState('')
  const [transOpen, setTransOpen] = useState(false)
  const [transType, setTransType] = useState<Transformer>('html')
  const selectedFolderRef = useRef<HTMLSelectElement>(null)
  const defaultFolderId = selected && (selected.folder_id ?? feedsById?.get(selected.feed_id)?.folder_id)

  const [transHtmlUrl, setTransHtmlUrl] = useState('')
  const [transHtmlTitle, setTransHtmlTitle] = useState('')
  const [transHtmlItems, setTransHtmlItems] = useState('')
  const [transHtmlItemTitle, setTransHtmlItemTitle] = useState('')
  const [transHtmlItemUrl, setTransHtmlItemUrl] = useState('')
  const [transHtmlItemUrlAttr, setTransHtmlItemUrlAttr] = useState('')
  const [transHtmlItemContent, setTransHtmlItemContent] = useState('')
  const [transHtmlItemDate, setTransHtmlItemDate] = useState('')
  const [transHtmlItemDateAttr, setTransHtmlItemDateAttr] = useState('')
  const transHtmlParams: Param[] = [
    {
      value: transHtmlUrl,
      setValue: setTransHtmlUrl,
      key: 'url',
      placeholder: 'https://example.com',
    },
    {
      value: transHtmlTitle,
      setValue: setTransHtmlTitle,
      key: 'title',
      desc: 'CSS selector targetting title of RSS',
      placeholder: 'title',
    },
    {
      value: transHtmlItems,
      setValue: setTransHtmlItems,
      key: 'items',
      desc: 'CSS selector targetting items',
      placeholder: 'html',
    },
    {
      value: transHtmlItemTitle,
      setValue: setTransHtmlItemTitle,
      key: 'item_title',
      desc: 'CSS selector targetting title of item',
      placeholder: 'same as item element',
    },
    {
      value: transHtmlItemUrl,
      setValue: setTransHtmlItemUrl,
      key: 'item_url',
      desc: 'CSS selector targetting URL of item',
      placeholder: 'same as item element',
    },
    {
      value: transHtmlItemUrlAttr,
      setValue: setTransHtmlItemUrlAttr,
      key: 'item_url_attr',
      desc: (
        <span>
          Attribute of <Code>item_url</Code> element as URL
        </span>
      ),
      placeholder: 'href',
    },
    {
      value: transHtmlItemContent,
      setValue: setTransHtmlItemContent,
      key: 'item_content',
      desc: 'CSS selector targetting content of item',
      placeholder: 'same as item element',
    },
    {
      value: transHtmlItemDate,
      setValue: setTransHtmlItemDate,
      key: 'item_date_published',
      desc: 'CSS selector targetting publication date of item',
      placeholder: 'same as item element',
    },
    {
      value: transHtmlItemDateAttr,
      setValue: setTransHtmlItemDateAttr,
      key: 'item_date_published_attr',
      desc: (
        <span>
          Attribute of <Code>item_date_published</Code> element as date
        </span>
      ),
      placeholder: 'element text',
    },
  ]

  const [transJsonUrl, setTransJsonUrl] = useState('')
  const [transJsonHomePageUrl, setTransJsonHomePageUrl] = useState('')
  const [transJsonTitle, setTransJsonTitle] = useState('')
  const [transJsonHeaders, setTransJsonHeaders] = useState('')
  const [transJsonItems, setTransJsonItems] = useState('')
  const [transJsonItemTitle, setTransJsonItemTitle] = useState('')
  const [transJsonItemUrl, setTransJsonItemUrl] = useState('')
  const [transJsonItemUrlPrefix, setTransJsonItemUrlPrefix] = useState('')
  const [transJsonItemContent, setTransJsonItemContent] = useState('')
  const [transJsonItemDate, setTransJsonItemDate] = useState('')
  const jsonPath = (
    <a
      style={{ color: 'inherit', textDecoration: 'underline' }}
      href="https://github.com/tidwall/gjson"
      target="_blank"
      rel="noopener noreferrer"
      referrerPolicy="no-referrer"
    >
      JSON path
    </a>
  )
  const transJsonParams: Param[] = [
    {
      value: transJsonUrl,
      setValue: setTransJsonUrl,
      key: 'url',
      placeholder: 'https://example.com',
    },
    {
      value: transJsonHomePageUrl,
      setValue: setTransJsonHomePageUrl,
      key: 'home_page_url',
      desc: 'Home page URL of RSS',
    },
    {
      value: transJsonTitle,
      setValue: setTransJsonTitle,
      key: 'title',
      desc: <span>{jsonPath} to title of RSS</span>,
    },
    {
      value: transJsonHeaders,
      setValue: setTransJsonHeaders,
      key: 'headers',
      desc: 'HTTP request headers in JSON format',
    },
    {
      value: transJsonItems,
      setValue: setTransJsonItems,
      key: 'items',
      desc: <span>{jsonPath} to items</span>,
      placeholder: 'entire JSON response',
    },
    {
      value: transJsonItemTitle,
      setValue: setTransJsonItemTitle,
      key: 'item_title',
      desc: <span>{jsonPath} to title of item</span>,
    },
    {
      value: transJsonItemUrl,
      setValue: setTransJsonItemUrl,
      key: 'item_url',
      desc: <span>{jsonPath} to URL of item</span>,
    },
    {
      value: transJsonItemUrlPrefix,
      setValue: setTransJsonItemUrlPrefix,
      key: 'item_url_prefix',
      desc: 'Optional prefix for URL',
    },
    {
      value: transJsonItemContent,
      setValue: setTransJsonItemContent,
      key: 'item_content',
      desc: <span>{jsonPath} to content of item</span>,
    },
    {
      value: transJsonItemDate,
      setValue: setTransJsonItemDate,
      key: 'item_date_published',
      desc: <span>{jsonPath} to publication date of item</span>,
    },
  ]

  const [js, setJs] = useState('')
  const jsParams: Param[] = [
    {
      value: js,
      setValue: setJs,
      key: 'script',
      desc: 'JavaScript',
      multiline: true,
    },
  ]

  const onConfirm = async () => {
    if (!selectedFolderRef.current) return
    if (!feedLink) throw new Error('Feed link is required')
    setLoading(true)
    try {
      const { feed, item_count } = await xfetch<{
        feed: Feed
        item_count: number
      }>('api/feeds', {
        method: 'POST',
        body: JSON.stringify({
          url: feedLink,
          folder_id: selectedFolderRef.current.value ? +selectedFolderRef.current.value : null,
        }),
      })
      setFeeds(feeds => feeds && [...feeds, feed].toSorted(compareTitle))
      updateStatus(status => {
        status?.state.set(feed.id, { unread: item_count, starred: 0 })
      })
      setSelected({ feed_id: feed.id })
      setFeedLink('')
      setIsOpen(false)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog
      title="New Feed"
      isOpen={isOpen}
      onClose={() => setIsOpen(false)}
      canEscapeKeyClose={false}
      onOpened={node => node.querySelector<HTMLInputElement>(`.${Classes.INPUT}`)?.focus()}
    >
      <DialogBody>
        <FormGroup label="URL" fill>
          <TextArea
            placeholder="https://example.com/feed"
            value={feedLink}
            spellCheck="false"
            autoResize
            fill
            onChange={evt => {
              const feedLink = evt.target.value
              setFeedLink(feedLink)
              const [scheme, url] = parseFeedLink(feedLink)
              if (scheme) {
                for (const { key, setValue } of {
                  html: transHtmlParams,
                  json: transJsonParams,
                  js: jsParams,
                }[scheme])
                  setValue(url.searchParams.get(key) ?? '')
                setTransType(scheme)
              }
            }}
            onKeyDown={async evt => {
              if (evt.key === 'Enter') {
                evt.preventDefault()
                await onConfirm()
              }
            }}
          />
        </FormGroup>
        <FormGroup label="Folder" fill>
          <HTMLSelect
            iconName="caret-down"
            options={[
              { value: '', label: '--' },
              ...(foldersWithFeeds ?? []).map(({ id, title }) => ({
                value: id,
                label: title,
              })),
            ]}
            defaultValue={defaultFolderId == null ? undefined : defaultFolderId}
            ref={selectedFolderRef}
            fill
          />
        </FormGroup>
        <FormGroup label="Transformer" style={{ marginBottom: '5px' }} fill>
          <div
            style={{
              borderRadius: 'var(--border-radius)',
              boxShadow: '0 0 0 1px rgb(from currentColor r g b / 20%)',
            }}
          >
            <TransformerSection
              style={{
                borderBottomLeftRadius: 0,
                borderBottomRightRadius: 0,
                boxShadow: 'none',
              }}
              type="html"
              title="HTML Transformer"
              params={transHtmlParams}
              isOpen={transOpen}
              setIsOpen={setTransOpen}
              curType={transType}
              setCurType={setTransType}
              setFeedLink={setFeedLink}
            />
            <Divider compact />
            <TransformerSection
              style={{
                borderRadius: 0,
                boxShadow: 'none',
              }}
              type="json"
              title="JSON Transformer"
              params={transJsonParams}
              isOpen={transOpen}
              setIsOpen={setTransOpen}
              curType={transType}
              setCurType={setTransType}
              setFeedLink={setFeedLink}
            />
            <Divider compact />
            <TransformerSection
              style={{
                borderTopLeftRadius: 0,
                borderTopRightRadius: 0,
                boxShadow: 'none',
              }}
              type="js"
              title="JavaScript"
              params={jsParams}
              isOpen={transOpen}
              setIsOpen={setTransOpen}
              curType={transType}
              setCurType={setTransType}
              setFeedLink={setFeedLink}
            />
          </div>
        </FormGroup>
      </DialogBody>
      <DialogFooter
        actions={<Button text="OK" loading={loading} intent={Intent.PRIMARY} onClick={onConfirm} fill />}
      />
    </Dialog>
  )
}

function TransformerSection({
  style,
  type,
  title,
  params,
  isOpen,
  setIsOpen,
  curType,
  setCurType,
  setFeedLink,
}: {
  style: CSSProperties
  type: Transformer
  title: string
  params: Param[]
  isOpen: boolean
  setIsOpen: Dispatch<SetStateAction<boolean>>
  curType: Transformer
  setCurType: Dispatch<SetStateAction<Transformer>>
  setFeedLink: Dispatch<SetStateAction<string>>
}) {
  const updateFeedLink = (i: number, value: string) =>
    setFeedLink(`rsslab://${curType}${stringify(params.with(i, { ...params[i], value }))}`)

  return (
    <Section
      style={style}
      title={title}
      titleRenderer={Span}
      collapseProps={{
        isOpen: isOpen && curType === type,
        onToggle: () => {
          if (isOpen && curType === type) {
            setIsOpen(false)
          } else {
            setCurType(type)
            setIsOpen(true)
          }
        },
        keepChildrenMounted: true,
      }}
      collapsible
      compact
    >
      <SectionCard>
        {params.map(({ value, setValue, key, desc, placeholder, multiline }, i) => (
          <FormGroup
            key={`${type}_${key}`}
            label={<Code>{key}</Code>}
            labelInfo={<span style={{ fontSize: '0.9em' }}>{desc}</span>}
            fill
          >
            {multiline ? (
              <TextArea
                fill
                autoResize
                spellCheck="false"
                size="small"
                wrap="off"
                value={value}
                style={{
                  minHeight: '10em',
                  fontFamily: 'var(--monospace)',
                }}
                onChange={evt => {
                  setValue(evt.target.value)
                  updateFeedLink(i, evt.target.value)
                }}
              />
            ) : (
              <InputGroup
                value={value}
                placeholder={placeholder}
                spellCheck="false"
                onValueChange={value => {
                  setValue(value)
                  updateFeedLink(i, value)
                }}
              />
            )}
          </FormGroup>
        ))}
        <AnchorButton
          text="Preview"
          href={`api/transform/${type}${stringify(params)}`}
          target="_blank"
          intent={Intent.PRIMARY}
          endIcon={<ExternalLink />}
          variant={ButtonVariant.OUTLINED}
          fill
        />
      </SectionCard>
    </Section>
  )
}

function Span(props: HTMLAttributes<HTMLSpanElement>) {
  return <span {...props} />
}

function stringify(params: Param[]) {
  return param(Object.fromEntries(params.filter(({ value }) => value).map(({ key, value }) => [key, value])))
}
