import { test, expect, Post } from '@teamkeel/testing'

test('create', async () => {
  const { object: post } = await Post.create({ title: 'apple' })

  expect.equal(post.title, 'apple')
})

test('update', async () => {
  const { object: post } = await Post.create({ title: 'star wars' })

  const { object: updatedPost } = await Post.update(post.id, { title: 'star wars sucks!' })

  expect.equal(updatedPost.title, 'star wars sucks!')
})

test('chained findOne', async () => {
  await Post.create({ title: 'apple' })
  await Post.create({ title: 'granny apple' })

  const { object: one } = await Post.where({
    title: {
      contains: 'apple'
    }
  }).findOne()

  expect.equal(one.title, 'apple')
})

test('simple all', async () => {
  await Post.create({ title: 'fruit' })
  await Post.create({ title: 'big fruit' })

  const { collection } = await Post.where({
    title: {
      contains: 'fruit'
    }
  }).all()

  expect.equal(collection.length, 2)
})

test('chained conditions with all', async () => {
  await Post.create({ title: 'melon' })
  await Post.create({ title: 'kiwi' })

  const { collection } = await Post.where({
    title: 'melon'
  }).orWhere({
    title: 'kiwi'
  }).all()

  expect.equal(collection.length, 2)
})

test('order', async () => {
  await Post.create({ title: 'abc' })
  await Post.create({ title: 'bcd' })

  const { collection } = await Post.where({
    title: {
      contains: 'bc'
    }
  }).order({
    title: 'desc'
  }).all()

  expect.equal(collection.length, 2)
  expect.equal(collection[0].title, 'abc')
})

test('sql', async () => {
  const sql = await Post.where({
    title: {
      contains: 'bc'
    }
  }).order({
    title: 'desc'
  }).sql({ asAst: false })

  expect.equal(sql, 'SELECT * FROM "post" WHERE ("post"."title" ILIKE $1) ORDER BY $2')
})

test('findMany', async () => {
  await Post.create({ title: 'io' })
  await Post.create({ title: 'iota' })

  const { collection } = await Post.findMany({
    title: {
      contains: 'io'
    }
  })

  expect.equal(collection.length, 2)
})

test('findOne', async () => {
  const { object: post } = await Post.create({ title: 'ghi' })
  await Post.create({ title: 'hij' })

  const { object } = await Post.findOne({
    title: {
      contains: 'hi'
    }
  })

  expect.equal(post.id, object.id)
})
