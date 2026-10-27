import { TranslationMessages } from 'react-admin';
import enUS from './en-US';
import { pageMessagesId } from './pages';
import { indonesianOverrides } from './indonesian-overrides';

type MessageTree = { [key: string]: string | MessageTree };

const isMessageTree = (value: unknown): value is MessageTree =>
  typeof value === 'object' && value !== null && !Array.isArray(value);

const deepMerge = (base: MessageTree, override: MessageTree): MessageTree => {
  const merged: MessageTree = { ...base };
  Object.entries(override).forEach(([key, value]) => {
    const current = merged[key];
    merged[key] =
      isMessageTree(current) && isMessageTree(value) ? deepMerge(current, value) : value;
  });
  return merged;
};

const indonesianMessages = deepMerge(
  enUS as unknown as MessageTree,
  {
    ...indonesianOverrides,
    pages: {
      ...(isMessageTree(indonesianOverrides.pages) ? indonesianOverrides.pages : {}),
      ...pageMessagesId,
    },
  } as MessageTree
) as unknown as TranslationMessages;

export default indonesianMessages;
