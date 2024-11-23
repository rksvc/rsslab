import {
  AnchorButton,
  Button,
  Code,
  Dialog,
  DialogBody,
  DialogFooter,
  FormGroup,
  HTMLSelect,
  InputGroup,
  Intent,
  RadioGroup,
  Section,
  SectionCard,
  TextArea,
} from '@blueprintjs/core'
import { type Dispatch, type HTMLAttributes, type SetStateAction, useRef, useState } from 'react'
import { ExternalLink } from 'react-feather'
import type { Feed, Folder, Selected, Status, Transformer } from './types.ts'
import { compareTitle, iconProps, length, parseFeedLink, xfetch } from './utils.ts'

type Param = {
  value: string
  setValue: Dispatch<SetStateAction<string>>
  key: string
  desc: string | JSX.Element
  parse?: (input: string) => any
  placeholder?: string
}

export function NewFeedDialog({
  isOpen,
  setIsOpen,
  defaultFolderId,
  darkTheme,

  folders,
  setFeeds,
  setStatus,
  setSelected,
}: {
  isOpen: boolean
  setIsOpen: Dispatch<SetStateAction<boolean>>
  defaultFolderId?: number | null
  darkTheme: boolean

  folders?: Folder[]
  setFeeds: Dispatch<SetStateAction<Feed[] | undefined>>
  setStatus: Dispatch<React.SetStateAction<Status | undefined>>
  setSelected: Dispatch<SetStateAction<Selected>>
}) {
  const [loading, setLoading] = useState(false)
  const [showGenerator, setShowGenerator] = useState(false)
  const [newFeedLink, setNewFeedLink] = useState('')
  const selectedFolderRef = useRef<HTMLSelectElement>(null)

  const [transType, setTransType] = useState<Transformer>('html')
  const [transUrl, setTransUrl] = useState('')

  const [transHtmlTitle, setTransHtmlTitle] = useState('')
  const [transHtmlItems, setTransHtmlItems] = useState('')
  const [transHtmlItemTitle, setTransHtmlItemTitle] = useState('')
  const [transHtmlItemUrl, setTransHtmlItemUrl] = useState('')
  const [transHtmlItemUrlAttr, setTransHtmlItemUrlAttr] = useState('')
  const [transHtmlItemContent, setTransHtmlItemContent] = useState('')
  const [transHtmlItemDate, setTransHtmlItemDate] = useState('')
  const [transHtmlItemDateAttr, setTransHtmlItemDateAttr] = useState('')

  const [transJsonHomePageUrl, setTransJsonHomePageUrl] = useState('')
  const [transJsonTitle, setTransJsonTitle] = useState('')
  const [transJsonHeaders, setTransJsonHeaders] = useState('')
  const [transJsonItems, setTransJsonItems] = useState('')
  const [transJsonItemTitle, setTransJsonItemTitle] = useState('')
  const [transJsonItemUrl, setTransJsonItemUrl] = useState('')
  const [transJsonItemUrlPrefix, setTransJsonItemUrlPrefix] = useState('')
  const [transJsonItemContent, setTransJsonItemContent] = useState('')
  const [transJsonItemDate, setTransJsonItemDate] = useState('')

  const transHtmlParams: Param[] = [
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
  const transJsonParams: Param[] = [
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
      desc: 'JSON path to title of RSS',
    },
    {
      value: transJsonHeaders,
      setValue: setTransJsonHeaders,
      key: 'headers',
      desc: 'HTTP request headers in JSON format',
      parse: (input: string) => {
        try {
          return JSON.parse(input)
        } catch {
          return null
        }
      },
    },
    {
      value: transJsonItems,
      setValue: setTransJsonItems,
      key: 'items',
      desc: 'JSON path to items',
      placeholder: 'entire JSON response',
    },
    {
      value: transJsonItemTitle,
      setValue: setTransJsonItemTitle,
      key: 'item_title',
      desc: 'JSON path to title of item',
    },
    {
      value: transJsonItemUrl,
      setValue: setTransJsonItemUrl,
      key: 'item_url',
      desc: 'JSON path to URL of item',
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
      desc: 'JSON path to content of item',
    },
    {
      value: transJsonItemDate,
      setValue: setTransJsonItemDate,
      key: 'item_date_published',
      desc: 'JSON path to publication date of item',
    },
  ]
  const transParamList = transType === 'html' ? transHtmlParams : transJsonParams
  const transParams = JSON.stringify({
    url: transUrl,
    ...Object.fromEntries(
      transParamList
        .map(({ key, value, parse }) => [key, parse ? parse(value) : value])
        .filter(([_, value]) => value),
    ),
  })
  const [isTypingTransParams, setIsTypingTransParams] = useState(false)
  const autoNewFeedLink = isTypingTransParams ? `${transType}:${transParams}` : newFeedLink

  const onConfirm = async () => {
    if (!selectedFolderRef.current) return
    if (!autoNewFeedLink) throw new Error('Feed link is required')
    setLoading(true)
    try {
      const { feed, item_count } = await xfetch<{
        feed: Feed
        item_count: number
      }>('api/feeds', {
        method: 'POST',
        body: JSON.stringify({
          url: autoNewFeedLink,
          folder_id: selectedFolderRef.current.value
            ? Number.parseInt(selectedFolderRef.current.value)
            : null,
        }),
      })
      setFeeds(feeds => feeds && [...feeds, feed].toSorted(compareTitle))
      setStatus(
        status =>
          status && {
            ...status,
            state: new Map([...status.state.entries(), [feed.id, { unread: item_count, starred: 0 }]]),
          },
      )
      setSelected({ feed_id: feed.id })
      setNewFeedLink('')
      setIsOpen(false)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog
      title="New Feed"
      className={darkTheme ? 'bp5-dark' : undefined}
      isOpen={isOpen}
      onClose={() => setIsOpen(false)}
      canEscapeKeyClose
      canOutsideClickClose
      onOpened={node => node.querySelector<HTMLInputElement>('.bp5-input')?.focus()}
    >
      <DialogBody>
        <div style={{ display: 'flex', flexDirection: 'column', gap: length(3) }}>
          <TextArea
            placeholder="https://example.com/feed"
            value={autoNewFeedLink}
            style={{ flexGrow: 1 }}
            spellCheck="false"
            autoResize
            onChange={evt => {
              const feedLink = evt.target.value
              setNewFeedLink(feedLink)
              setIsTypingTransParams(false)
              const [scheme, link] = parseFeedLink(feedLink)
              if (scheme)
                try {
                  const paramList = scheme === 'html' ? transHtmlParams : transJsonParams
                  const params: Record<string, any> = JSON.parse(link)
                  for (const { key, setValue } of paramList) {
                    const value = params[key] || ''
                    setValue(typeof value === 'string' ? value : JSON.stringify(value))
                  }
                  setTransUrl(params.url ?? '')
                  setTransType(scheme)
                } catch {}
            }}
            onKeyDown={async evt => {
              if (evt.key === 'Enter') {
                evt.preventDefault()
                await onConfirm()
              }
            }}
          />
          <HTMLSelect
            iconName="caret-down"
            options={[
              { value: '', label: '--' },
              ...(folders ?? []).map(({ id, title }) => ({
                value: id,
                label: title,
              })),
            ]}
            defaultValue={defaultFolderId == null ? undefined : defaultFolderId}
            ref={selectedFolderRef}
            fill
          />
          <Section
            title="Feed Generator"
            titleRenderer={Span}
            collapseProps={{
              isOpen: showGenerator,
              onToggle: () => setShowGenerator(showGenerator => !showGenerator),
              keepChildrenMounted: true,
              transitionDuration: 200,
            }}
            collapsible
            compact
          >
            <SectionCard>
              <div style={{ textAlign: 'center' }}>
                <RadioGroup
                  selectedValue={transType}
                  onChange={evt => setTransType(evt.currentTarget.value as Transformer)}
                  options={[
                    { value: 'html', label: 'HTML Transformer' },
                    { value: 'json', label: 'JSON Transformer' },
                  ]}
                  inline
                />
              </div>
              {[
                {
                  value: transUrl,
                  setValue: setTransUrl,
                  key: 'url',
                  desc: undefined,
                  placeholder: 'https://example.com',
                },
                ...transParamList,
              ].map(({ value, setValue, key, desc, placeholder }) => (
                <FormGroup
                  key={`${transType}_${key}`}
                  label={<Code>{key}</Code>}
                  labelFor={`${transType}_${key}`}
                  labelInfo={<span style={{ fontSize: '0.9em' }}>{desc}</span>}
                  fill
                >
                  <InputGroup
                    value={value}
                    id={`${transType}_${key}`}
                    placeholder={placeholder}
                    spellCheck="false"
                    onValueChange={value => {
                      setValue(value)
                      setIsTypingTransParams(true)
                    }}
                  />
                </FormGroup>
              ))}
              <AnchorButton
                text="Preview"
                href={`api/transform/${transType}/${encodeURIComponent(transParams)}`}
                target="_blank"
                intent={Intent.PRIMARY}
                rightIcon={<ExternalLink {...iconProps} />}
                outlined
                fill
              />
            </SectionCard>
          </Section>
        </div>
      </DialogBody>
      <DialogFooter
        actions={<Button text="OK" loading={loading} intent={Intent.PRIMARY} onClick={onConfirm} fill />}
      />
    </Dialog>
  )
}

function Span(props: HTMLAttributes<HTMLSpanElement>) {
  return <span {...props} />
}
