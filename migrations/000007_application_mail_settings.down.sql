UPDATE applications
SET settings = settings - 'mail'
WHERE settings ? 'mail';
